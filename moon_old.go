package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// moonOldInstruction is one instruction from the variable-length SNX format
// used by MOON. DVD. It is deliberately separate from the later 12-byte LCSE
// format used by ONE Vista.
type moonOldInstruction struct {
	offset       int
	opcode       byte
	bytes        []byte
	words        []uint16
	strings      []string
	stringRanges []moonOldByteRange
	expr         []byte
	target       uint16
	textSpecial  bool
}

type moonOldByteRange struct {
	start int
	end   int
}

type moonOldReader struct {
	data []byte
	pos  int
}

func (r *moonOldReader) need(n int) error {
	if n < 0 || r.pos+n > len(r.data) {
		return fmt.Errorf("truncated instruction at 0x%04X", r.pos)
	}
	return nil
}

func (r *moonOldReader) byte() (byte, error) {
	if err := r.need(1); err != nil {
		return 0, err
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}

func (r *moonOldReader) word() (uint16, error) {
	if err := r.need(2); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos : r.pos+2])
	r.pos += 2
	return v, nil
}

func (r *moonOldReader) raw(n int) ([]byte, error) {
	if err := r.need(n); err != nil {
		return nil, err
	}
	v := append([]byte(nil), r.data[r.pos:r.pos+n]...)
	r.pos += n
	return v, nil
}

func (r *moonOldReader) cstring() (string, error) {
	end := bytes.IndexByte(r.data[r.pos:], 0)
	if end < 0 {
		return "", fmt.Errorf("unterminated string at 0x%04X", r.pos)
	}
	raw := r.data[r.pos : r.pos+end]
	r.pos += end + 1
	text, err := s2u(raw)
	if err != nil {
		return "", fmt.Errorf("decode Shift-JIS string at 0x%04X: %w", r.pos-end-1, err)
	}
	return text, nil
}

func (r *moonOldReader) text() (string, bool, error) {
	end := bytes.IndexByte(r.data[r.pos:], 0)
	if end < 0 {
		return "", false, fmt.Errorf("unterminated TEXT at 0x%04X", r.pos)
	}
	if end == 0 {
		r.pos++
		return "", false, nil
	}
	raw := r.data[r.pos : r.pos+end]
	r.pos += end + 1
	// Ordinary MOON text ends in 05 00. The special TEXT ...^ form omits
	// the 05 suffix; assembler variants may retain or omit the 02 marker.
	special := raw[len(raw)-1] != 0x05
	if !special {
		raw = raw[:len(raw)-1]
	}
	text, err := s2u(raw)
	if err != nil {
		return "", special, fmt.Errorf("decode Shift-JIS TEXT at 0x%04X: %w", r.pos-end-1, err)
	}
	return text, special, nil
}

func parseMoonOldSNX(data []byte) ([]moonOldInstruction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty MOON SNX")
	}
	r := &moonOldReader{data: data}
	var out []moonOldInstruction

	readBytes := func(ins *moonOldInstruction, n int) error {
		v, err := r.raw(n)
		if err == nil {
			ins.bytes = v
		}
		return err
	}
	readWords := func(ins *moonOldInstruction, n int) error {
		for i := 0; i < n; i++ {
			v, err := r.word()
			if err != nil {
				return err
			}
			ins.words = append(ins.words, v)
		}
		return nil
	}
	readString := func(ins *moonOldInstruction) error {
		start := r.pos
		v, err := r.cstring()
		if err == nil {
			ins.strings = append(ins.strings, v)
			ins.stringRanges = append(ins.stringRanges, moonOldByteRange{start: start, end: r.pos - 1})
		}
		return err
	}

	for r.pos < len(data) {
		start := r.pos
		op, err := r.byte()
		if err != nil {
			return nil, err
		}
		ins := moonOldInstruction{offset: start, opcode: op}

		switch op {
		case 0x00, 0x01, 0x04, 0x08, 0x0C:
			err = readWords(&ins, 2)
		case 0x24, 0x25, 0x2C, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x39, 0x51:
			err = readBytes(&ins, 1)
		case 0x10: // JMP
			ins.target, err = r.word()
		case 0x11: // IF: expression bytecode terminated by 00, then target
			for {
				var token byte
				token, err = r.byte()
				if err != nil {
					break
				}
				ins.expr = append(ins.expr, token)
				if token == 0 {
					break
				}
				if token == 0x15 || token == 0x16 || token == 0x17 {
					var operand []byte
					operand, err = r.raw(2)
					if err != nil {
						break
					}
					ins.expr = append(ins.expr, operand...)
				}
			}
			if err == nil {
				ins.target, err = r.word()
			}
		case 0x15: // TEXT
			var prefix byte
			prefix, err = r.byte()
			if err == nil && prefix != 0x01 {
				err = fmt.Errorf("unsupported TEXT prefix %02X at 0x%04X", prefix, start)
			}
			if err == nil {
				ins.bytes = append(ins.bytes, prefix)
				// Ordinary text has a second 02 marker. A trailing ^ in the
				// assembler source omits it (used by TEST.SNX).
				if r.pos < len(r.data) && r.data[r.pos] == 0x02 {
					ins.bytes = append(ins.bytes, 0x02)
					r.pos++
				}
			}
			if err == nil {
				textStart := r.pos
				var text string
				text, ins.textSpecial, err = r.text()
				if err == nil {
					ins.strings = append(ins.strings, text)
					textEnd := r.pos - 1
					if !ins.textSpecial && textEnd > textStart {
						textEnd--
					}
					ins.stringRanges = append(ins.stringRanges, moonOldByteRange{start: textStart, end: textEnd})
				}
			}
		case 0x17: // SELECT: flag + N strings + empty-string terminator
			err = readWords(&ins, 1)
			for err == nil {
				choiceStart := r.pos
				var choice string
				choice, err = r.cstring()
				if err != nil || choice == "" {
					break
				}
				ins.strings = append(ins.strings, choice)
				ins.stringRanges = append(ins.stringRanges, moonOldByteRange{start: choiceStart, end: r.pos - 1})
			}
		case 0x18, 0x19, 0x20, 0x21, 0x40, 0x50:
			err = readString(&ins)
		case 0x28, 0x36, 0x41:
			err = readWords(&ins, 1)
		case 0x2B: // PLAY_SOUND byte, string, byte
			err = readBytes(&ins, 1)
			if err == nil {
				err = readString(&ins)
			}
			if err == nil {
				var tail []byte
				tail, err = r.raw(1)
				ins.bytes = append(ins.bytes, tail...)
			}
		case 0x2D, 0x4E: // string, byte
			err = readString(&ins)
			if err == nil {
				err = readBytes(&ins, 1)
			}
		case 0x2E: // mode, three strings, then records [byte + 4 words], 00 terminated
			err = readBytes(&ins, 1)
			for i := 0; i < 3 && err == nil; i++ {
				err = readString(&ins)
			}
			for err == nil {
				var recordType byte
				recordType, err = r.byte()
				if err != nil {
					break
				}
				ins.expr = append(ins.expr, recordType)
				if recordType == 0 {
					break
				}
				var record []byte
				record, err = r.raw(8)
				ins.expr = append(ins.expr, record...)
			}
		case 0x2F:
			err = readString(&ins)
			if err == nil {
				err = readWords(&ins, 6)
			}
		case 0x37, 0x38:
			err = readBytes(&ins, 6)
		case 0x3A, 0x4D:
			err = readWords(&ins, 4)
		case 0x3B, 0x3C, 0x3F:
			err = readBytes(&ins, 2)
		case 0x3D: // byte, string
			err = readBytes(&ins, 1)
			if err == nil {
				err = readString(&ins)
			}
		case 0x3E: // two bytes, string
			err = readBytes(&ins, 2)
			if err == nil {
				err = readString(&ins)
			}
		case 0x42:
			err = readWords(&ins, 8)
		case 0x52:
			err = readWords(&ins, 2)
		case 0x56:
			err = readBytes(&ins, 1)
			if err == nil {
				err = readWords(&ins, 2)
			}
		case 0x06, 0x16, 0x1A, 0x45, 0x46, 0x4F, 0x57:
			// no operands
		case 0xFF:
			// The Japanese INIT.SNX retains an unreachable 93-byte fragment
			// after its first EOF. The official MOON Kit drops it as well.
			// Preserve its presence for diagnostics, but do not treat the valid
			// script as a failed conversion.
			ins.bytes = append(ins.bytes, r.data[r.pos:]...)
			out = append(out, ins)
			return out, nil
		default:
			return nil, fmt.Errorf("unknown MOON opcode 0x%02X at 0x%04X", op, start)
		}
		if err != nil {
			return nil, fmt.Errorf("MOON opcode 0x%02X at 0x%04X: %w", op, start, err)
		}
		out = append(out, ins)
	}
	return nil, fmt.Errorf("MOON SNX has no EOF opcode")
}

// MOON's original assembler can lay out and wrap the translated text when
// every character occupies one source byte. The GUI therefore feeds it these
// otherwise-unused control bytes for French accents. Once assembly is done,
// only bytes inside parsed string operands are converted to the 0xA1-0xAD
// values understood by lcse_hook.dll. Opcode and operand bytes are untouched.
var moonAccentSentinelToByte = map[byte]byte{
	0x02: 0xA1, // e acute
	0x03: 0xA2, // e grave
	0x04: 0xA3, // c cedilla
	0x06: 0xA4, // a grave
	0x07: 0xA5, // a circumflex
	0x08: 0xA6, // u circumflex
	0x0E: 0xA7, // o circumflex
	0x0F: 0xA8, // e circumflex
	0x10: 0xA9, // i circumflex
	0x11: 0xAA, // u grave
	0x12: 0xAB, // e diaeresis
	0x13: 0xAC, // i diaeresis
	0x14: 0xAD, // u diaeresis
}

func applyMoonAccentSentinels(data []byte) ([]byte, int, error) {
	instructions, err := parseMoonOldSNX(data)
	if err != nil {
		return nil, 0, err
	}
	out := append([]byte(nil), data...)
	changed := 0
	for _, ins := range instructions {
		for _, span := range ins.stringRanges {
			for pos := span.start; pos < span.end; pos++ {
				if replacement, ok := moonAccentSentinelToByte[out[pos]]; ok {
					out[pos] = replacement
					changed++
				}
			}
		}
	}
	return out, changed, nil
}

func cmdMoonAccents(inputPath, outputPath string) error {
	info, err := os.Stat(inputPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if outputPath == "" {
			return fmt.Errorf("output directory required for moon-accents batch mode")
		}
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return err
		}
		files, err := filepath.Glob(filepath.Join(inputPath, "*.[sS][nN][xX]"))
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return fmt.Errorf("no MOON SNX in %s", inputPath)
		}
		total := 0
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			patched, count, err := applyMoonAccentSentinels(data)
			if err != nil {
				return fmt.Errorf("%s: %w", filepath.Base(file), err)
			}
			if err := os.WriteFile(filepath.Join(outputPath, filepath.Base(file)), patched, 0644); err != nil {
				return err
			}
			total += count
		}
		fmt.Printf("[INFO] %d MOON SNX, %d accents finalises -> %s\n", len(files), total, outputPath)
		return nil
	}

	if outputPath == "" {
		return fmt.Errorf("output SNX required (input is never modified in place)")
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	patched, count, err := applyMoonAccentSentinels(data)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(outputPath, patched, 0644); err != nil {
		return err
	}
	fmt.Printf("[INFO] %s: %d accents finalises -> %s\n", filepath.Base(inputPath), count, outputPath)
	return nil
}

func moonQuoted(s string) string {
	s = moonReadableSpacing(s)
	for strings.Contains(s, " \x01") {
		s = strings.ReplaceAll(s, " \x01", "\x01")
	}
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\x01", "\\n")
	return "'" + s + "'"
}

func moonText(s string) string {
	s = moonReadableSpacing(s)
	for strings.Contains(s, " \x01") {
		s = strings.ReplaceAll(s, " \x01", "\x01")
	}
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\x01", "\\n")
	return s
}

func moonReadableSpacing(s string) string {
	hasLatin := false
	hasJapanese := false
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasLatin = true
		}
		if (r >= 0x3040 && r <= 0x30FF) || (r >= 0x3400 && r <= 0x9FFF) {
			hasJapanese = true
		}
	}
	if hasLatin && !hasJapanese {
		return strings.ReplaceAll(s, "\u3000", " ")
	}
	return s
}

func moonByteList(values []byte) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("%02X", value)
	}
	return strings.Join(parts, " ")
}

func moonWordList(values []uint16) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("%04X", value)
	}
	return strings.Join(parts, " ")
}

func renderMoonOldSNX(name string, instructions []moonOldInstruction) (string, error) {
	targets := map[int]bool{}
	starts := map[int]bool{}
	for _, ins := range instructions {
		starts[ins.offset] = true
		if ins.opcode == 0x10 || ins.opcode == 0x11 {
			targets[int(ins.target)] = true
		}
	}
	var targetOffsets []int
	for target := range targets {
		if !starts[target] {
			return "", fmt.Errorf("branch target 0x%04X is not an instruction boundary", target)
		}
		targetOffsets = append(targetOffsets, target)
	}
	sort.Ints(targetOffsets)
	labels := map[int]string{}
	for i, target := range targetOffsets {
		labels[target] = fmt.Sprintf("LABEL%d", i)
	}

	var out strings.Builder
	fmt.Fprintf(&out, "# MOON SCRIPT FILE %q\r\n", strings.ToUpper(filepath.Base(name)))
	out.WriteString("# Direct disassembly by lcse-tool; UTF-8 with BOM.\r\n\r\n")

	line := func(command string) {
		out.WriteString(command)
		out.WriteString("\r\n")
	}
	for _, ins := range instructions {
		if label := labels[ins.offset]; label != "" {
			fmt.Fprintf(&out, "\r\n:%s\r\n", label)
		}
		var command string
		switch ins.opcode {
		case 0x00, 0x01, 0x04, 0x08, 0x0C:
			command = fmt.Sprintf("X_%02X %s", ins.opcode, moonWordList(ins.words))
		case 0x06:
			command = "RETURN"
		case 0x10:
			command = "JMP ." + labels[int(ins.target)]
		case 0x11:
			command = "IF " + moonByteList(ins.expr) + " ." + labels[int(ins.target)]
		case 0x15:
			command = "TEXT " + moonText(ins.strings[0])
			if ins.textSpecial {
				command += "^"
			}
		case 0x16:
			command = "X_16"
		case 0x17:
			var choices []string
			for _, choice := range ins.strings {
				choices = append(choices, moonQuoted(choice))
			}
			command = fmt.Sprintf("SELECT %04X", ins.words[0])
			if len(choices) > 0 {
				command += " " + strings.Join(choices, " ")
			}
		case 0x18:
			command = "GOTO_SCRIPT " + moonQuoted(ins.strings[0])
		case 0x19:
			command = "GOSUB_SCRIPT " + moonQuoted(ins.strings[0])
		case 0x1A:
			command = "RETURN"
		case 0x20:
			command = "BACKGROUND " + moonQuoted(ins.strings[0])
		case 0x21:
			command = "SPRITE " + moonQuoted(ins.strings[0])
		case 0x24, 0x25, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x39, 0x51:
			command = fmt.Sprintf("X_%02X %s", ins.opcode, moonByteList(ins.bytes))
		case 0x28, 0x36, 0x41:
			command = fmt.Sprintf("X_%02X %s", ins.opcode, moonWordList(ins.words))
		case 0x2B:
			command = fmt.Sprintf("PLAY_SOUND %02X %s %02X", ins.bytes[0], moonQuoted(ins.strings[0]), ins.bytes[1])
		case 0x2C:
			command = "STOP_SOUND " + moonByteList(ins.bytes)
		case 0x2D, 0x4E:
			command = fmt.Sprintf("X_%02X %s %02X", ins.opcode, moonQuoted(ins.strings[0]), ins.bytes[0])
		case 0x2E:
			var args []string
			args = append(args, fmt.Sprintf("%02X", ins.bytes[0]))
			for _, value := range ins.strings {
				args = append(args, moonQuoted(value))
			}
			for pos := 0; pos < len(ins.expr); {
				recordType := ins.expr[pos]
				args = append(args, fmt.Sprintf("%02X", recordType))
				pos++
				if recordType == 0 {
					break
				}
				for i := 0; i < 4; i++ {
					value := binary.LittleEndian.Uint16(ins.expr[pos : pos+2])
					args = append(args, fmt.Sprintf("%04X", value))
					pos += 2
				}
			}
			command = "X_2E " + strings.Join(args, " ")
		case 0x2F:
			command = "X_2F " + moonQuoted(ins.strings[0]) + " " + moonWordList(ins.words)
		case 0x37, 0x38, 0x3B, 0x3C, 0x3F:
			command = fmt.Sprintf("X_%02X %s", ins.opcode, moonByteList(ins.bytes))
		case 0x3A, 0x42, 0x4D, 0x52:
			command = fmt.Sprintf("X_%02X %s", ins.opcode, moonWordList(ins.words))
		case 0x3D:
			command = fmt.Sprintf("X_3D %02X %s", ins.bytes[0], moonQuoted(ins.strings[0]))
		case 0x3E:
			command = fmt.Sprintf("X_3E %02X %02X %s", ins.bytes[0], ins.bytes[1], moonQuoted(ins.strings[0]))
		case 0x40:
			command = "SETSTATUS " + moonQuoted(ins.strings[0])
		case 0x45, 0x46, 0x4F, 0x57:
			command = fmt.Sprintf("X_%02X", ins.opcode)
		case 0x50:
			command = "X_50 " + moonQuoted(ins.strings[0])
		case 0x56:
			command = fmt.Sprintf("X_56 %02X %s", ins.bytes[0], moonWordList(ins.words))
		case 0xFF:
			command = "EOF"
		default:
			return "", fmt.Errorf("cannot render MOON opcode 0x%02X", ins.opcode)
		}
		line(command)
	}
	return out.String(), nil
}

func cmdMoonOldSNX2TXTData(sp, op string, data []byte) error {
	instructions, err := parseMoonOldSNX(data)
	if err != nil {
		return err
	}
	text, err := renderMoonOldSNX(sp, instructions)
	if err != nil {
		return err
	}
	if op == "" {
		op = strings.TrimSuffix(sp, filepath.Ext(sp)) + ".txt"
	}
	if dir := filepath.Dir(op); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	payload := append([]byte{0xEF, 0xBB, 0xBF}, []byte(text)...)
	if err := os.WriteFile(op, payload, 0644); err != nil {
		return err
	}
	textCount := 0
	trailingCount := 0
	for _, ins := range instructions {
		if ins.opcode == 0x15 {
			textCount++
		}
		if ins.opcode == 0xFF {
			trailingCount += len(ins.bytes)
		}
	}
	if trailingCount > 0 {
		fmt.Printf("[WARN] %s: %d octets orphelins apres EOF ignores (comportement du MOON Kit officiel)\n",
			filepath.Base(sp), trailingCount)
	}
	fmt.Printf("[INFO] %s: MOON ancien, %d instructions (%d TEXT) -> %s [UTF-8]\n",
		filepath.Base(sp), len(instructions), textCount, op)
	return nil
}
