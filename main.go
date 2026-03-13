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

func sjisToUTF8(b []byte) (string, error) {
	r := transform.NewReader(bytes.NewReader(b), japanese.ShiftJIS.NewDecoder())
	out, err := io.ReadAll(r)
	return string(out), err
}
func utf8ToSJIS(s string) ([]byte, error) {
	r := transform.NewReader(strings.NewReader(s), japanese.ShiftJIS.NewEncoder())
	return io.ReadAll(r)
}
func expandKey(b byte) uint32 {
	k := uint32(b); return k | (k << 8) | (k << 16) | (k << 24)
}
func xorBytes(data []byte, key byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data { out[i] = b ^ key }
	return out
}
func autoDetectKey(lstPath string) (byte, error) {
	f, err := os.Open(lstPath); if err != nil { return 0, err }
	defer f.Close()
	fi, _ := f.Stat()
	var buf [4]byte; f.Read(buf[:])
	key := buf[3]; if key == 0 { return 0, fmt.Errorf("cannot auto-detect key") }
	count := binary.LittleEndian.Uint32(buf[:]) ^ expandKey(key)
	if int(count)*lstEntrySize+4 == int(fi.Size()) { return key, nil }
	return 0, fmt.Errorf("auto-detect failed")
}

// ─── LST ─────────────────────────────────────────────────────────────────────

type LSTEntry struct {
	Offset, Size uint32; Name string; TypeID uint32; Ext string
}

func parseLST(lstPath string, intKey byte) ([]LSTEntry, error) {
	lstData, err := os.ReadFile(lstPath); if err != nil { return nil, err }
	xorKey := expandKey(intKey)
	count := binary.LittleEndian.Uint32(lstData[0:4]) ^ xorKey
	if int(count)*lstEntrySize+4 != len(lstData) { return nil, fmt.Errorf("LST mismatch (key=0x%02x)", intKey) }
	entries := make([]LSTEntry, count)
	pos := 4
	for i := uint32(0); i < count; i++ {
		offset := binary.LittleEndian.Uint32(lstData[pos:pos+4]) ^ xorKey
		size := binary.LittleEndian.Uint32(lstData[pos+4:pos+8]) ^ xorKey
		pos += 8
		nb := make([]byte, filenameSize); copy(nb, lstData[pos:pos+filenameSize]); pos += filenameSize
		nl := 0
		for j := 0; j < filenameSize; j++ {
			if nb[j] == 0 { break }; if nb[j] != intKey { nb[j] ^= intKey }; nl = j + 1
		}
		name, _ := sjisToUTF8(nb[:nl]); if name == "" { name = string(nb[:nl]) }
		typeID := binary.LittleEndian.Uint32(lstData[pos : pos+4]); pos += 4
		ext := typeExtMap[typeID]; if ext == "" { ext = fmt.Sprintf("%d", typeID) }
		entries[i] = LSTEntry{offset, size, name, typeID, ext}
	}
	return entries, nil
}
func resolveKey(lstPath string, ko, so int) (byte, byte, error) {
	var ik, sk byte
	if ko >= 0 { ik = byte(ko) } else {
		d, e := autoDetectKey(lstPath); if e != nil { return 0, 0, e }; ik = d
	}
	if so >= 0 { sk = byte(so) } else { sk = ik + 1 }
	fmt.Printf("[INFO] Key: 0x%02X, SNX key: 0x%02X\n", ik, sk); return ik, sk, nil
}

// ─── Archive commands ────────────────────────────────────────────────────────

func cmdUnpack(pakPath, outDir string, ko, so int) error {
	ik, sk, err := resolveKey(pakPath+".lst", ko, so); if err != nil { return err }
	entries, err := parseLST(pakPath+".lst", ik); if err != nil { return err }
	pf, err := os.Open(pakPath); if err != nil { return err }; defer pf.Close()
	os.MkdirAll(outDir, 0755)
	fmt.Printf("[INFO] Extracting %d files...\n", len(entries))
	for _, e := range entries {
		fn := e.Name + "." + e.Ext; data := make([]byte, e.Size)
		pf.ReadAt(data, int64(e.Offset))
		if e.Ext == "snx" { data = xorBytes(data, sk) }
		os.WriteFile(filepath.Join(outDir, fn), data, 0644)
	}
	fmt.Printf("[INFO] Done! %d files -> %s\n", len(entries), outDir); return nil
}

func cmdPatch(origPak, patchDir, outPath string, ko, so int) error {
	ik, sk, err := resolveKey(origPak+".lst", ko, so); if err != nil { return err }
	entries, err := parseLST(origPak+".lst", ik); if err != nil { return err }
	op, _ := os.Open(origPak); defer op.Close()
	repl := map[string]string{}
	filepath.Walk(patchDir, func(p string, i os.FileInfo, e error) error {
		if e != nil || i.IsDir() { return e }; repl[strings.ToUpper(filepath.Base(p))] = p; return nil
	})
	of, _ := os.Create(outPath); defer of.Close()
	lf, _ := os.Create(outPath + ".lst"); defer lf.Close()
	xk := expandKey(ik); var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(len(entries))^xk); lf.Write(buf[:])
	patched := 0
	for _, e := range entries {
		fn := e.Name + "." + e.Ext
		var data []byte
		if rp, ok := repl[strings.ToUpper(fn)]; ok {
			data, _ = os.ReadFile(rp)
			if e.Ext == "snx" { data = xorBytes(data, sk) }
			fmt.Printf("[PATCH] %s\n", fn); patched++
		} else {
			data = make([]byte, e.Size); op.ReadAt(data, int64(e.Offset))
		}
		po, _ := of.Seek(0, io.SeekCurrent)
		binary.LittleEndian.PutUint32(buf[:], uint32(po)^xk); lf.Write(buf[:])
		binary.LittleEndian.PutUint32(buf[:], uint32(len(data))^xk); lf.Write(buf[:])
		sn, _ := utf8ToSJIS(e.Name); if sn == nil { sn = []byte(e.Name) }
		nb := make([]byte, filenameSize)
		for i, b := range sn { if i < filenameSize { nb[i] = b ^ ik } }
		lf.Write(nb)
		binary.LittleEndian.PutUint32(buf[:], e.TypeID); lf.Write(buf[:])
		of.Write(data)
	}
	fmt.Printf("[INFO] Patched %d/%d -> %s\n", patched, len(entries), outPath); return nil
}

func cmdPack(srcDir, outPath string, ko, so int) error {
	ik := byte(0x01); sk := byte(0x02)
	if ko >= 0 { ik = byte(ko); sk = ik + 1 }; if so >= 0 { sk = byte(so) }
	var files []string
	filepath.Walk(srcDir, func(p string, i os.FileInfo, e error) error {
		if e != nil || i.IsDir() { return e }; files = append(files, p); return nil
	}); sort.Strings(files)
	pf, _ := os.Create(outPath); defer pf.Close()
	lf, _ := os.Create(outPath + ".lst"); defer lf.Close()
	xk := expandKey(ik); var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(len(files))^xk); lf.Write(buf[:])
	for _, fp := range files {
		data, _ := os.ReadFile(fp)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fp), "."))
		bn := strings.TrimSuffix(filepath.Base(fp), filepath.Ext(fp))
		po, _ := pf.Seek(0, io.SeekCurrent)
		binary.LittleEndian.PutUint32(buf[:], uint32(po)^xk); lf.Write(buf[:])
		binary.LittleEndian.PutUint32(buf[:], uint32(len(data))^xk); lf.Write(buf[:])
		sn, _ := utf8ToSJIS(bn); if sn == nil { sn = []byte(bn) }
		nb := make([]byte, filenameSize)
		for i, b := range sn { nb[i] = b ^ ik }; lf.Write(nb)
		tid := extTypeMap[ext]
		binary.LittleEndian.PutUint32(buf[:], tid); lf.Write(buf[:])
		if ext == "snx" { data = xorBytes(data, sk) }; pf.Write(data)
	}; return nil
}

// ─── SNX ─────────────────────────────────────────────────────────────────────

type SNXEntry struct {
	Offset    uint32 // offset in string table
	SlotSize  uint32 // total slot: 4 (len prefix) + slen
	RawBytes  []byte // slen bytes from table (includes null)
	SJISClean []byte // without null and \x02\x03
	HasCtrl   bool   // had \x02\x03 suffix
	IsDialogue bool
}

type SNXFile struct {
	H0, H1   uint32
	Bytecode []byte
	Entries  []SNXEntry
}

func parseSNX(data []byte) (*SNXFile, error) {
	if len(data) < 8 { return nil, fmt.Errorf("SNX too small") }
	h0 := binary.LittleEndian.Uint32(data[0:4])
	h1 := binary.LittleEndian.Uint32(data[4:8])
	if int(h1) > len(data) { return nil, fmt.Errorf("str table %d > file %d", h1, len(data)) }
	ss := uint32(len(data)) - h1
	bc := make([]byte, ss); copy(bc, data[:ss])
	var entries []SNXEntry
	pos := uint32(0)
	for pos+4 <= h1 {
		slen := binary.LittleEndian.Uint32(data[ss+pos : ss+pos+4])
		if slen == 0 || pos+4+slen > h1 { break }
		raw := make([]byte, slen); copy(raw, data[ss+pos+4:ss+pos+4+slen])
		nn := raw; if len(nn) > 0 && nn[len(nn)-1] == 0 { nn = nn[:len(nn)-1] }
		hc := false; cl := nn
		if len(cl) >= 2 && cl[len(cl)-2] == 0x02 && cl[len(cl)-1] == 0x03 { hc = true; cl = cl[:len(cl)-2] }
		text, _ := sjisToUTF8(cl)
		entries = append(entries, SNXEntry{
			Offset: pos, SlotSize: 4 + slen, RawBytes: raw, SJISClean: cl,
			HasCtrl: hc, IsDialogue: containsCJK(text),
		})
		pos += 4 + slen
	}
	return &SNXFile{H0: h0, H1: h1, Bytecode: bc, Entries: entries}, nil
}

func containsCJK(s string) bool {
	for _, r := range s {
		if (r >= 0x3000 && r <= 0x9FFF) || (r >= 0xF900 && r <= 0xFAFF) { return true }
	}; return false
}

// isStringRef checks if position i in bytecode is a string table reference.
// Pattern 1: [0x11][0x02][str_offset] - "push string" instruction
// Pattern 2: [0x0F|0x10|0x15][str_offset] - direct string opcodes
func isStringRef(bc []byte, i int) bool {
	if i < 8 { return false }
	prev := binary.LittleEndian.Uint32(bc[i-4 : i])
	// Pattern 1: prev=0x02, prev-prev=0x11
	if prev == 0x02 && i >= 12 {
		pp := binary.LittleEndian.Uint32(bc[i-8 : i-4])
		if pp == 0x11 { return true }
	}
	// Pattern 2: direct string opcodes
	if prev == 0x0F || prev == 0x10 || prev == 0x15 { return true }
	return false
}

// ─── snx2txt ─────────────────────────────────────────────────────────────────

func cmdSNX2TXT(snxPath, outPath string) error {
	data, _ := os.ReadFile(snxPath)
	snx, err := parseSNX(data); if err != nil { return err }
	if outPath == "" { outPath = strings.TrimSuffix(snxPath, filepath.Ext(snxPath)) + ".txt" }
	f, _ := os.Create(outPath); defer f.Close()
	w := bufio.NewWriter(f); defer w.Flush()
	w.WriteString("# LCSE SNX: " + filepath.Base(snxPath) + "\r\n")
	w.WriteString("# INDEX\\tOFFSET\\tTYPE\\tTEXT (Shift-JIS)\r\n#\r\n")
	dlg := 0
	for i, e := range snx.Entries {
		t := "RES"
		if e.IsDialogue { t = "TXT"; dlg++ } else {
			ok := false; for _, b := range e.SJISClean { if b >= 0x20 && b <= 0x7E { ok = true } }
			if !ok && len(e.SJISClean) <= 2 { t = "CTL" }
		}
		fmt.Fprintf(w, "%d\t0x%04X\t%s\t", i, e.Offset, t)
		w.Write(e.SJISClean)
		w.WriteString("\r\n")
	}
	fmt.Printf("[INFO] %s: %d strings (%d dialogue) -> %s\n", filepath.Base(snxPath), len(snx.Entries), dlg, outPath)
	return nil
}

// ─── txt2snx ─────────────────────────────────────────────────────────────────

func cmdTXT2SNX(txtPath, snxPath, outPath string) error {
	origData, _ := os.ReadFile(snxPath)
	snx, err := parseSNX(origData); if err != nil { return err }
	txtData, _ := os.ReadFile(txtPath)
	if outPath == "" { outPath = strings.TrimSuffix(snxPath, filepath.Ext(snxPath)) + "_patched" + filepath.Ext(snxPath) }

	// Parse translations from SJIS text file
	textMap := map[int][]byte{}
	for _, line := range bytes.Split(txtData, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		if len(line) == 0 || line[0] == '#' { continue }
		parts := bytes.SplitN(line, []byte("\t"), 4)
		if len(parts) < 4 { continue }
		var idx int
		if _, err := fmt.Sscanf(string(parts[0]), "%d", &idx); err != nil { continue }
		if len(parts[3]) > 0 { textMap[idx] = append([]byte{}, parts[3]...) }
	}

	// Phase 1: Build new string table, keeping entries at original offsets when possible
	newStrTable := make([]byte, snx.H1) // start with original size
	copy(newStrTable, origData[len(origData)-int(snx.H1):]) // copy original table

	oldToNew := map[uint32]uint32{} // only for relocated entries
	changed := 0
	overflow := []int{}

	for i, e := range snx.Entries {
		newSJIS, ok := textMap[i]
		if !ok { continue } // no change

		// Build new entry data
		var newData []byte
		newData = append(newData, newSJIS...)
		if e.HasCtrl { newData = append(newData, 0x02, 0x03) }
		newData = append(newData, 0x00) // null terminator

		newSlen := uint32(len(newData))
		origSlen := e.SlotSize - 4 // original data size (from length prefix)

		if newSlen <= origSlen {
			// Fits in original slot: write in-place with padding
			binary.LittleEndian.PutUint32(newStrTable[e.Offset:], newSlen)
			copy(newStrTable[e.Offset+4:], newData)
			// Zero-pad remaining
			for j := e.Offset + 4 + newSlen; j < e.Offset + e.SlotSize; j++ {
				newStrTable[j] = 0
			}
			changed++
		} else {
			// Overflow: append at end of table, keep original at original position
			overflow = append(overflow, i)
			newOffset := uint32(len(newStrTable))
			var lenBuf [4]byte
			binary.LittleEndian.PutUint32(lenBuf[:], newSlen)
			newStrTable = append(newStrTable, lenBuf[:]...)
			newStrTable = append(newStrTable, newData...)
			oldToNew[e.Offset] = newOffset
			changed++
		}
	}

	// Phase 2: Update bytecode - ONLY for relocated entries, ONLY at verified string ref positions
	bc := make([]byte, len(snx.Bytecode))
	copy(bc, snx.Bytecode)

	if len(overflow) > 0 {
		refsUpdated := 0
		for i := 8; i+4 <= len(bc); i += 4 {
			val := binary.LittleEndian.Uint32(bc[i : i+4])
			if newOff, ok := oldToNew[val]; ok {
				if isStringRef(bc, i) {
					binary.LittleEndian.PutUint32(bc[i:i+4], newOff)
					refsUpdated++
				}
			}
		}
		// Update string table size in header
		binary.LittleEndian.PutUint32(bc[4:8], uint32(len(newStrTable)))
		fmt.Printf("[INFO] %d entries relocated, %d refs updated\n", len(overflow), refsUpdated)
	}

	// Write output
	var out bytes.Buffer
	out.Write(bc)
	out.Write(newStrTable)
	os.WriteFile(outPath, out.Bytes(), 0644)
	fmt.Printf("[INFO] %d strings modifiees -> %s\n", changed, outPath)
	return nil
}

// ─── Batch ───────────────────────────────────────────────────────────────────

func cmdSNX2TXTBatch(dir, outDir string) error {
	if outDir == "" { outDir = dir + "_txt" }; os.MkdirAll(outDir, 0755)
	m, _ := filepath.Glob(filepath.Join(dir, "*.[sS][nN][xX]"))
	if len(m) == 0 { return fmt.Errorf("no SNX in %s", dir) }
	fmt.Printf("[INFO] Batch: %d files\n", len(m))
	for _, f := range m {
		b := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		cmdSNX2TXT(f, filepath.Join(outDir, b+".txt"))
	}; return nil
}
func cmdTXT2SNXBatch(td, sd, od string) error {
	if od == "" { od = sd + "_patched" }; os.MkdirAll(od, 0755)
	m, _ := filepath.Glob(filepath.Join(td, "*.txt"))
	if len(m) == 0 { return fmt.Errorf("no TXT in %s", td) }
	for _, f := range m {
		b := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		sp := filepath.Join(sd, b+".SNX")
		if _, e := os.Stat(sp); e != nil { sp = filepath.Join(sd, b+".snx")
			if _, e := os.Stat(sp); e != nil { continue } }
		cmdTXT2SNX(f, sp, filepath.Join(od, b+".snx"))
	}; return nil
}

// ─── Main ────────────────────────────────────────────────────────────────────

func usage() {
	fmt.Fprintf(os.Stderr, `lcse-tool v4 - LC-ScriptEngine (One ~Kagayaku Kisetsu e~)

ARCHIVE:
  lcse-tool unpack <lcsebody> [output_dir]
  lcse-tool patch <lcsebody_original> <dossier_patches> <sortie>
  lcse-tool pack <dossier> <sortie>

SCRIPTS:
  lcse-tool snx2txt <fichier.snx|dossier> [sortie]
  lcse-tool txt2snx <texte.txt> <original.snx> [sortie.snx]
  lcse-tool txt2snx-batch <dossier_txt> <dossier_snx> [dossier_sortie]

OPTIONS:  --key <hex>  --snxkey <hex>  (default: auto-detect)
`)
}

func main() {
	args := os.Args[1:]; ko, so := -1, -1; var pos []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key": if i+1<len(args) { var k int; fmt.Sscanf(args[i+1],"%x",&k); ko=k; i++ }
		case "--snxkey": if i+1<len(args) { var k int; fmt.Sscanf(args[i+1],"%x",&k); so=k; i++ }
		case "-h","--help": usage(); os.Exit(0)
		default: pos = append(pos, args[i])
		}
	}
	if len(pos) < 1 { usage(); os.Exit(1) }
	cmd, ca := pos[0], pos[1:]
	var err error
	switch cmd {
	case "unpack","u":
		if len(ca)<1 { usage(); os.Exit(1) }
		od := ca[0]+"_extracted"; if len(ca)>=2 { od=ca[1] }
		err = cmdUnpack(ca[0], od, ko, so)
	case "patch":
		if len(ca)<3 { usage(); os.Exit(1) }
		err = cmdPatch(ca[0], ca[1], ca[2], ko, so)
	case "pack","p":
		if len(ca)<2 { usage(); os.Exit(1) }
		err = cmdPack(ca[0], ca[1], ko, so)
	case "snx2txt","s2t":
		if len(ca)<1 { usage(); os.Exit(1) }
		o:=""; if len(ca)>=2 { o=ca[1] }
		if inf,_:=os.Stat(ca[0]); inf!=nil && inf.IsDir() { err=cmdSNX2TXTBatch(ca[0],o) } else { err=cmdSNX2TXT(ca[0],o) }
	case "txt2snx","t2s":
		if len(ca)<2 { usage(); os.Exit(1) }
		o:=""; if len(ca)>=3 { o=ca[2] }
		err = cmdTXT2SNX(ca[0], ca[1], o)
	case "txt2snx-batch","t2s-batch":
		if len(ca)<2 { usage(); os.Exit(1) }
		o:=""; if len(ca)>=3 { o=ca[2] }
		err = cmdTXT2SNXBatch(ca[0], ca[1], o)
	default: fmt.Fprintf(os.Stderr, "Commande inconnue: %s\n", cmd); usage(); os.Exit(1)
	}
	if err != nil { fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err); os.Exit(1) }
}
