package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const (
	filenameSize = 0x40
	lstEntrySize = filenameSize + 12
)

var typeExtMap = map[uint32]string{1: "snx", 2: "bmp", 3: "png", 4: "wav", 5: "ogg"}
var extTypeMap = map[string]uint32{"snx": 1, "bmp": 2, "png": 3, "wav": 4, "ogg": 5}

// ─── Encoding ────────────────────────────────────────────────────────────────

func sjisToUTF8(b []byte) (string, error) {
	r := transform.NewReader(bytes.NewReader(b), japanese.ShiftJIS.NewDecoder())
	out, err := io.ReadAll(r)
	return string(out), err
}

func utf8ToSJIS(s string) ([]byte, error) {
	r := transform.NewReader(strings.NewReader(s), japanese.ShiftJIS.NewEncoder())
	return io.ReadAll(r)
}

func sjisBytes(s string) []byte {
	b, err := utf8ToSJIS(s)
	if err != nil {
		return []byte(s)
	}
	return b
}

// ─── XOR ─────────────────────────────────────────────────────────────────────

func expandKey(b byte) uint32 {
	k := uint32(b)
	return k | (k << 8) | (k << 16) | (k << 24)
}

func xorBytes(data []byte, key byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key
	}
	return out
}

func autoDetectKey(lstPath string) (byte, error) {
	f, err := os.Open(lstPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fi, _ := f.Stat()
	var buf [4]byte
	f.Read(buf[:])
	key := buf[3]
	if key == 0 {
		return 0, fmt.Errorf("cannot auto-detect key")
	}
	count := binary.LittleEndian.Uint32(buf[:]) ^ expandKey(key)
	if int(count)*lstEntrySize+4 == int(fi.Size()) {
		return key, nil
	}
	return 0, fmt.Errorf("auto-detect failed")
}

// ─── LST parsing ─────────────────────────────────────────────────────────────

type LSTEntry struct {
	Offset, Size uint32
	Name         string
	TypeID       uint32
	Ext          string
}

func parseLST(lstPath string, intKey byte) ([]LSTEntry, error) {
	lstData, err := os.ReadFile(lstPath)
	if err != nil {
		return nil, err
	}
	xorKey := expandKey(intKey)
	count := binary.LittleEndian.Uint32(lstData[0:4]) ^ xorKey
	if int(count)*lstEntrySize+4 != len(lstData) {
		return nil, fmt.Errorf("LST size mismatch (key=0x%02x)", intKey)
	}
	entries := make([]LSTEntry, count)
	pos := 4
	for i := uint32(0); i < count; i++ {
		offset := binary.LittleEndian.Uint32(lstData[pos:pos+4]) ^ xorKey
		size := binary.LittleEndian.Uint32(lstData[pos+4:pos+8]) ^ xorKey
		pos += 8
		nameBytes := make([]byte, filenameSize)
		copy(nameBytes, lstData[pos:pos+filenameSize])
		pos += filenameSize
		nameLen := 0
		for j := 0; j < filenameSize; j++ {
			if nameBytes[j] == 0 {
				break
			}
			if nameBytes[j] != intKey {
				nameBytes[j] ^= intKey
			}
			nameLen = j + 1
		}
		name, _ := sjisToUTF8(nameBytes[:nameLen])
		if name == "" {
			name = string(nameBytes[:nameLen])
		}
		typeID := binary.LittleEndian.Uint32(lstData[pos : pos+4])
		pos += 4
		ext := typeExtMap[typeID]
		if ext == "" {
			ext = fmt.Sprintf("%d", typeID)
		}
		entries[i] = LSTEntry{Offset: offset, Size: size, Name: name, TypeID: typeID, Ext: ext}
	}
	return entries, nil
}

func resolveKey(lstPath string, keyOvr, snxOvr int) (byte, byte, error) {
	var intKey, snxKey byte
	if keyOvr >= 0 {
		intKey = byte(keyOvr)
	} else {
		det, err := autoDetectKey(lstPath)
		if err != nil {
			return 0, 0, fmt.Errorf("auto-detect: %w (use --key)", err)
		}
		intKey = det
	}
	if snxOvr >= 0 {
		snxKey = byte(snxOvr)
	} else {
		snxKey = intKey + 1
	}
	fmt.Printf("[INFO] Key: 0x%02X, SNX key: 0x%02X\n", intKey, snxKey)
	return intKey, snxKey, nil
}

// ─── unpack ──────────────────────────────────────────────────────────────────

func cmdUnpack(pakPath, outDir string, keyOvr, snxOvr int) error {
	lstPath := pakPath + ".lst"
	intKey, snxKey, err := resolveKey(lstPath, keyOvr, snxOvr)
	if err != nil {
		return err
	}
	entries, err := parseLST(lstPath, intKey)
	if err != nil {
		return err
	}
	pakFile, err := os.Open(pakPath)
	if err != nil {
		return fmt.Errorf("opening PAK: %w", err)
	}
	defer pakFile.Close()
	os.MkdirAll(outDir, 0755)
	fmt.Printf("[INFO] Extracting %d files...\n", len(entries))
	for _, e := range entries {
		fname := e.Name + "." + e.Ext
		data := make([]byte, e.Size)
		if _, err := pakFile.ReadAt(data, int64(e.Offset)); err != nil {
			return fmt.Errorf("reading %s: %w", fname, err)
		}
		if e.Ext == "snx" {
			data = xorBytes(data, snxKey)
		}
		if err := os.WriteFile(filepath.Join(outDir, fname), data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", fname, err)
		}
	}
	fmt.Printf("[INFO] Done! %d files extracted to %s\n", len(entries), outDir)
	return nil
}

// ─── patch (replace files in existing archive) ──────────────────────────────

func cmdPatch(origPakPath, patchDir, outPath string, keyOvr, snxOvr int) error {
	lstPath := origPakPath + ".lst"
	intKey, snxKey, err := resolveKey(lstPath, keyOvr, snxOvr)
	if err != nil {
		return err
	}
	entries, err := parseLST(lstPath, intKey)
	if err != nil {
		return err
	}
	origPak, err := os.Open(origPakPath)
	if err != nil {
		return err
	}
	defer origPak.Close()

	// Find replacement files
	replacements := make(map[string]string) // normalized name -> file path
	filepath.Walk(patchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		replacements[strings.ToUpper(filepath.Base(path))] = path
		return nil
	})

	outPak, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outPak.Close()

	outLst, err := os.Create(outPath + ".lst")
	if err != nil {
		return err
	}
	defer outLst.Close()

	xorKey := expandKey(intKey)
	var buf [4]byte

	// Write entry count
	binary.LittleEndian.PutUint32(buf[:], uint32(len(entries))^xorKey)
	outLst.Write(buf[:])

	patched := 0
	for _, e := range entries {
		fname := e.Name + "." + e.Ext
		fnameUpper := strings.ToUpper(fname)

		var data []byte

		if replPath, ok := replacements[fnameUpper]; ok {
			// Use replacement file
			data, err = os.ReadFile(replPath)
			if err != nil {
				return fmt.Errorf("reading patch %s: %w", replPath, err)
			}
			fmt.Printf("[PATCH] %s <- %s\n", fname, filepath.Base(replPath))
			patched++
			// XOR-encrypt SNX for storage
			if e.Ext == "snx" {
				data = xorBytes(data, snxKey)
			}
		} else {
			// Copy from original archive (already encrypted)
			data = make([]byte, e.Size)
			if _, err := origPak.ReadAt(data, int64(e.Offset)); err != nil {
				return fmt.Errorf("reading original %s: %w", fname, err)
			}
		}

		// Current output offset
		pakOffset, _ := outPak.Seek(0, io.SeekCurrent)

		// Write LST entry: offset
		binary.LittleEndian.PutUint32(buf[:], uint32(pakOffset)^xorKey)
		outLst.Write(buf[:])

		// size
		binary.LittleEndian.PutUint32(buf[:], uint32(len(data))^xorKey)
		outLst.Write(buf[:])

		// filename (re-encode from original name)
		sjisName, _ := utf8ToSJIS(e.Name)
		if sjisName == nil {
			sjisName = []byte(e.Name)
		}
		nameBuf := make([]byte, filenameSize)
		for i, b := range sjisName {
			if i >= filenameSize {
				break
			}
			nameBuf[i] = b ^ intKey
		}
		outLst.Write(nameBuf)

		// type
		binary.LittleEndian.PutUint32(buf[:], e.TypeID)
		outLst.Write(buf[:])

		// Write data to PAK
		outPak.Write(data)
	}

	fmt.Printf("[INFO] Patched %d/%d files -> %s (+%s.lst)\n", patched, len(entries), outPath, outPath)
	return nil
}

// ─── pack (from scratch) ────────────────────────────────────────────────────

func cmdPack(srcDir, outPath string, keyOvr, snxOvr int) error {
	intKey := byte(0x01)
	snxKey := byte(0x02)
	if keyOvr >= 0 {
		intKey = byte(keyOvr)
		snxKey = intKey + 1
	}
	if snxOvr >= 0 {
		snxKey = byte(snxOvr)
	}
	fmt.Printf("[INFO] Key: 0x%02X, SNX key: 0x%02X\n", intKey, snxKey)

	var files []string
	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)

	pakFile, _ := os.Create(outPath)
	defer pakFile.Close()
	lstFile, _ := os.Create(outPath + ".lst")
	defer lstFile.Close()

	xorKey := expandKey(intKey)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(len(files))^xorKey)
	lstFile.Write(buf[:])

	for _, fpath := range files {
		data, _ := os.ReadFile(fpath)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fpath), "."))
		baseName := strings.TrimSuffix(filepath.Base(fpath), filepath.Ext(fpath))
		pakOffset, _ := pakFile.Seek(0, io.SeekCurrent)

		binary.LittleEndian.PutUint32(buf[:], uint32(pakOffset)^xorKey)
		lstFile.Write(buf[:])
		binary.LittleEndian.PutUint32(buf[:], uint32(len(data))^xorKey)
		lstFile.Write(buf[:])

		sjisName, _ := utf8ToSJIS(baseName)
		if sjisName == nil {
			sjisName = []byte(baseName)
		}
		nameBuf := make([]byte, filenameSize)
		for i, b := range sjisName {
			nameBuf[i] = b ^ intKey
		}
		lstFile.Write(nameBuf)

		typeID, ok := extTypeMap[ext]
		if !ok {
			fmt.Printf("[WARN] unknown ext %s, skipping %s\n", ext, fpath)
			continue
		}
		binary.LittleEndian.PutUint32(buf[:], typeID)
		lstFile.Write(buf[:])

		if ext == "snx" {
			data = xorBytes(data, snxKey)
		}
		pakFile.Write(data)
	}
	fmt.Println("[INFO] Pack complete!")
	return nil
}

// ─── SNX ─────────────────────────────────────────────────────────────────────

type SNXStringEntry struct {
	TableOffset uint32
	RawBytes    []byte // raw bytes from table (includes null)
	Text        string // display text (UTF-8, cleaned)
	SJISClean   []byte // Shift-JIS text without \x02\x03 and null
	IsDialogue  bool
	HasCtrlSuffix bool // had \x02\x03 at the end
}

type SNXFile struct {
	CodeCount, StringTableSize uint32
	Bytecode                   []byte
	StringEntries              []SNXStringEntry
}

func parseSNX(data []byte) (*SNXFile, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("SNX too small")
	}
	codeCount := binary.LittleEndian.Uint32(data[0:4])
	strTblSz := binary.LittleEndian.Uint32(data[4:8])
	if int(strTblSz) > len(data) {
		return nil, fmt.Errorf("string table size %d > file %d", strTblSz, len(data))
	}
	strStart := uint32(len(data)) - strTblSz
	bc := make([]byte, strStart)
	copy(bc, data[:strStart])

	var entries []SNXStringEntry
	pos := uint32(0)
	for pos+4 <= strTblSz {
		slen := binary.LittleEndian.Uint32(data[strStart+pos : strStart+pos+4])
		if slen == 0 || pos+4+slen > strTblSz {
			break
		}
		raw := make([]byte, slen)
		copy(raw, data[strStart+pos+4:strStart+pos+4+slen])

		// Strip null for processing
		noNull := raw
		if len(noNull) > 0 && noNull[len(noNull)-1] == 0 {
			noNull = noNull[:len(noNull)-1]
		}

		// Check for \x02\x03 suffix
		hasCtrl := false
		clean := noNull
		if len(clean) >= 2 && clean[len(clean)-2] == 0x02 && clean[len(clean)-1] == 0x03 {
			hasCtrl = true
			clean = clean[:len(clean)-2]
		}

		text, _ := sjisToUTF8(clean)
		if text == "" {
			text = fmt.Sprintf("<hex:%x>", clean)
		}

		entries = append(entries, SNXStringEntry{
			TableOffset:   pos,
			RawBytes:      raw,
			Text:          text,
			SJISClean:     clean,
			IsDialogue:    containsCJK(text),
			HasCtrlSuffix: hasCtrl,
		})
		pos += 4 + slen
	}

	return &SNXFile{CodeCount: codeCount, StringTableSize: strTblSz, Bytecode: bc, StringEntries: entries}, nil
}

func containsCJK(s string) bool {
	for _, r := range s {
		if (r >= 0x3000 && r <= 0x9FFF) || (r >= 0xF900 && r <= 0xFAFF) {
			return true
		}
	}
	return false
}

// ─── snx2txt: export as Shift-JIS TSV (4 columns) ──────────────────────────

func cmdSNX2TXT(snxPath, outPath string) error {
	data, err := os.ReadFile(snxPath)
	if err != nil {
		return err
	}
	snx, err := parseSNX(data)
	if err != nil {
		return err
	}
	if outPath == "" {
		outPath = strings.TrimSuffix(snxPath, filepath.Ext(snxPath)) + ".txt"
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write Shift-JIS encoded output
	w := bufio.NewWriter(f)
	defer w.Flush()

	// Header (ASCII, safe in Shift-JIS)
	w.WriteString("# LCSE SNX Script: " + filepath.Base(snxPath) + "\r\n")
	w.WriteString("# INDEX\\tOFFSET\\tTYPE\\tTEXT (Shift-JIS)\r\n")
	w.WriteString("# TXT=dialogue RES=resource CTL=control\r\n")
	w.WriteString("# Remplacez le texte japonais par votre traduction\r\n")
	w.WriteString("#\r\n")

	dlg := 0
	for i, e := range snx.StringEntries {
		t := "RES"
		if e.IsDialogue {
			t = "TXT"
			dlg++
		} else if len(e.Text) <= 2 {
			ok := false
			for _, r := range e.Text {
				if r >= 0x20 && r <= 0x7E {
					ok = true
				}
			}
			if !ok {
				t = "CTL"
			}
		}

		// Write: INDEX\tOFFSET\tTYPE\t[SJIS text]
		header := fmt.Sprintf("%d\t0x%04X\t%s\t", i, e.TableOffset, t)
		w.Write([]byte(header))
		w.Write(e.SJISClean) // Direct Shift-JIS bytes
		w.WriteString("\r\n")
	}

	fmt.Printf("[INFO] %s: %d strings (%d dialogue) -> %s [Shift-JIS]\n",
		filepath.Base(snxPath), len(snx.StringEntries), dlg, outPath)
	return nil
}

// ─── txt2snx: read Shift-JIS TSV, rebuild SNX ──────────────────────────────

func cmdTXT2SNX(txtPath, snxPath, outPath string) error {
	origData, err := os.ReadFile(snxPath)
	if err != nil {
		return err
	}
	snx, err := parseSNX(origData)
	if err != nil {
		return err
	}

	// Read the TXT file (Shift-JIS encoded)
	txtData, err := os.ReadFile(txtPath)
	if err != nil {
		return err
	}

	// Parse lines
	textMap := make(map[int][]byte) // index -> SJIS bytes
	lines := bytes.Split(txtData, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		// Split by tab (max 4 parts)
		parts := bytes.SplitN(line, []byte("\t"), 4)
		if len(parts) < 4 {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(string(parts[0]), "%d", &idx); err != nil {
			continue
		}
		sjisText := parts[3]
		if len(sjisText) > 0 {
			textMap[idx] = append([]byte{}, sjisText...) // copy
		}
	}

	if outPath == "" {
		outPath = strings.TrimSuffix(snxPath, filepath.Ext(snxPath)) + "_patched" + filepath.Ext(snxPath)
	}

	// Build new string table
	oldToNew := make(map[uint32]uint32)
	var newStr bytes.Buffer

	changed := 0
	for i, e := range snx.StringEntries {
		oldToNew[e.TableOffset] = uint32(newStr.Len())

		var sd []byte
		if newText, ok := textMap[i]; ok {
			sd = newText
			// Re-add \x02\x03 suffix if original had it
			if e.HasCtrlSuffix {
				sd = append(sd, 0x02, 0x03)
			}
			// Check if it actually changed
			if !bytes.Equal(sd, e.SJISClean) && !(e.HasCtrlSuffix && bytes.Equal(sd, append(e.SJISClean, 0x02, 0x03))) {
				changed++
			}
		} else {
			// Keep original (without null)
			sd = e.RawBytes
			if len(sd) > 0 && sd[len(sd)-1] == 0 {
				sd = sd[:len(sd)-1]
			}
		}

		// Write: [uint32 len+1] [data] [\0]
		var lb [4]byte
		binary.LittleEndian.PutUint32(lb[:], uint32(len(sd)+1))
		newStr.Write(lb[:])
		newStr.Write(sd)
		newStr.WriteByte(0)
	}

	// Update bytecode references
	bc := make([]byte, len(snx.Bytecode))
	copy(bc, snx.Bytecode)
	for i := 8; i+4 <= len(bc); i += 4 {
		val := binary.LittleEndian.Uint32(bc[i : i+4])
		if nv, ok := oldToNew[val]; ok && val != nv {
			binary.LittleEndian.PutUint32(bc[i:i+4], nv)
		}
	}
	binary.LittleEndian.PutUint32(bc[4:8], uint32(newStr.Len()))

	var out bytes.Buffer
	out.Write(bc)
	out.Write(newStr.Bytes())
	if err := os.WriteFile(outPath, out.Bytes(), 0644); err != nil {
		return err
	}
	fmt.Printf("[INFO] %d strings modifiees -> %s\n", changed, outPath)
	return nil
}

// ─── Batch ───────────────────────────────────────────────────────────────────

func cmdSNX2TXTBatch(dir, outDir string) error {
	if outDir == "" {
		outDir = dir + "_txt"
	}
	os.MkdirAll(outDir, 0755)
	matches, _ := filepath.Glob(filepath.Join(dir, "*.snx"))
	up, _ := filepath.Glob(filepath.Join(dir, "*.SNX"))
	matches = append(matches, up...)
	if len(matches) == 0 {
		return fmt.Errorf("no SNX in %s", dir)
	}
	fmt.Printf("[INFO] Batch: %d files\n", len(matches))
	for _, m := range matches {
		base := strings.TrimSuffix(filepath.Base(m), filepath.Ext(m))
		cmdSNX2TXT(m, filepath.Join(outDir, base+".txt"))
	}
	return nil
}

func cmdTXT2SNXBatch(txtDir, snxDir, outDir string) error {
	if outDir == "" {
		outDir = snxDir + "_patched"
	}
	os.MkdirAll(outDir, 0755)
	matches, _ := filepath.Glob(filepath.Join(txtDir, "*.txt"))
	if len(matches) == 0 {
		return fmt.Errorf("no TXT in %s", txtDir)
	}
	fmt.Printf("[INFO] Batch: %d files\n", len(matches))
	for _, m := range matches {
		base := strings.TrimSuffix(filepath.Base(m), filepath.Ext(m))
		sp := filepath.Join(snxDir, base+".SNX")
		if _, err := os.Stat(sp); err != nil {
			sp = filepath.Join(snxDir, base+".snx")
			if _, err := os.Stat(sp); err != nil {
				continue
			}
		}
		cmdTXT2SNX(m, sp, filepath.Join(outDir, base+".snx"))
	}
	return nil
}

// ─── Main ────────────────────────────────────────────────────────────────────

func usage() {
	fmt.Fprintf(os.Stderr, `lcse-tool - LC-ScriptEngine toolkit (One ~Kagayaku Kisetsu e~)

ARCHIVE:
  lcse-tool unpack <lcsebody> [output_dir]
      Extraire tous les fichiers (cle auto-detectee)

  lcse-tool patch <lcsebody_original> <dossier_patches> <sortie>
      Remplacer des fichiers dans l'archive (garde les PNG intacts)

  lcse-tool pack <dossier> <sortie>
      Recreer une archive de zero (tous les fichiers requis)

SCRIPTS:
  lcse-tool snx2txt <fichier.snx|dossier> [sortie]
      SNX -> fichier texte Shift-JIS editable

  lcse-tool txt2snx <texte.txt> <original.snx> [sortie.snx]
      Reinjecter le texte modifie dans un SNX

  lcse-tool txt2snx-batch <dossier_txt> <dossier_snx> [dossier_sortie]
      Reinjection en batch

OPTIONS:
  --key <hex>      Forcer la cle XOR (defaut: auto-detect)
  --snxkey <hex>   Forcer la cle SNX (defaut: key+1)

WORKFLOW:
  1. lcse-tool unpack lcsebody1 extracted/
  2. lcse-tool snx2txt extracted/ scripts/
  3. Editer les .txt (remplacer le japonais par le francais)
  4. lcse-tool txt2snx-batch scripts/ extracted/ patched/
  5. lcse-tool patch lcsebody1 patched/ lcsebody1_fr
`)
}

func main() {
	args := os.Args[1:]
	keyOvr, snxOvr := -1, -1
	var pos []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key":
			if i+1 < len(args) {
				var k int
				fmt.Sscanf(args[i+1], "%x", &k)
				keyOvr = k
				i++
			}
		case "--snxkey":
			if i+1 < len(args) {
				var k int
				fmt.Sscanf(args[i+1], "%x", &k)
				snxOvr = k
				i++
			}
		case "-h", "--help":
			usage()
			os.Exit(0)
		default:
			pos = append(pos, args[i])
		}
	}
	if len(pos) < 1 {
		usage()
		os.Exit(1)
	}
	cmd, ca := pos[0], pos[1:]
	var err error
	switch cmd {
	case "unpack", "u":
		if len(ca) < 1 {
			fmt.Fprintln(os.Stderr, "need <lcsebody>")
			os.Exit(1)
		}
		od := ca[0] + "_extracted"
		if len(ca) >= 2 {
			od = ca[1]
		}
		err = cmdUnpack(ca[0], od, keyOvr, snxOvr)

	case "patch":
		if len(ca) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: lcse-tool patch <original> <patch_dir> <output>")
			os.Exit(1)
		}
		err = cmdPatch(ca[0], ca[1], ca[2], keyOvr, snxOvr)

	case "pack", "p":
		if len(ca) < 2 {
			fmt.Fprintln(os.Stderr, "need <dir> <output>")
			os.Exit(1)
		}
		err = cmdPack(ca[0], ca[1], keyOvr, snxOvr)

	case "snx2txt", "s2t":
		if len(ca) < 1 {
			fmt.Fprintln(os.Stderr, "need <file|dir>")
			os.Exit(1)
		}
		o := ""
		if len(ca) >= 2 {
			o = ca[1]
		}
		info, _ := os.Stat(ca[0])
		if info != nil && info.IsDir() {
			err = cmdSNX2TXTBatch(ca[0], o)
		} else {
			err = cmdSNX2TXT(ca[0], o)
		}

	case "txt2snx", "t2s":
		if len(ca) < 2 {
			fmt.Fprintln(os.Stderr, "need <txt> <snx>")
			os.Exit(1)
		}
		o := ""
		if len(ca) >= 3 {
			o = ca[2]
		}
		err = cmdTXT2SNX(ca[0], ca[1], o)

	case "txt2snx-batch", "t2s-batch":
		if len(ca) < 2 {
			fmt.Fprintln(os.Stderr, "need <txt_dir> <snx_dir>")
			os.Exit(1)
		}
		o := ""
		if len(ca) >= 3 {
			o = ca[2]
		}
		err = cmdTXT2SNXBatch(ca[0], ca[1], o)

	default:
		fmt.Fprintf(os.Stderr, "Commande inconnue: %s\n", cmd)
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}
