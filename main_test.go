package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMoonLSTDecodesShiftJISFilename(t *testing.T) {
	const key byte = 0xCC
	const filename = "CG_メッセージ窓２.TGF"
	const normalizedFilename = "CG_メッセージ窓２.tgf"

	rawName, err := u2s(filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(rawName) > moonFnSz {
		t.Fatalf("test filename is too long: %d bytes", len(rawName))
	}

	data := make([]byte, 4+moonLstESz)
	binary.LittleEndian.PutUint32(data[0:4], uint32(1)^exK(key))
	binary.LittleEndian.PutUint32(data[4:8], uint32(0)^exK(key))
	binary.LittleEndian.PutUint32(data[8:12], uint32(123)^exK(key))
	for i, value := range rawName {
		data[12+i] = value ^ key
	}

	lst := filepath.Join(t.TempDir(), "moon.lst")
	if err := os.WriteFile(lst, data, 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := pLST(lst, key, lstMoon)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if got := entries[0].FN(); got != normalizedFilename {
		t.Fatalf("expected %q, got %q", normalizedFilename, got)
	}
}

func TestMoonPatchPreservesShiftJISFilename(t *testing.T) {
	const key byte = 0xCC
	const filename = "選択中.TGF"
	const normalizedFilename = "選択中.tgf"

	root := t.TempDir()
	archive := filepath.Join(root, "moon")
	if err := os.WriteFile(archive, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	rawName, err := u2s(filename)
	if err != nil {
		t.Fatal(err)
	}
	lstData := make([]byte, 4+moonLstESz)
	binary.LittleEndian.PutUint32(lstData[0:4], uint32(1)^exK(key))
	binary.LittleEndian.PutUint32(lstData[4:8], uint32(0)^exK(key))
	binary.LittleEndian.PutUint32(lstData[8:12], uint32(4)^exK(key))
	for i, value := range rawName {
		lstData[12+i] = value ^ key
	}
	if err := os.WriteFile(archive+".lst", lstData, 0644); err != nil {
		t.Fatal(err)
	}
	patches := filepath.Join(root, "empty_patch")
	if err := os.Mkdir(patches, 0755); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(root, "moon_rebuilt")
	if err := cmdPatch(archive, patches, output, -1, -1); err != nil {
		t.Fatal(err)
	}
	if rebuiltArchive, err := os.ReadFile(output); err != nil || string(rebuiltArchive) != "test" {
		t.Fatalf("unpatched resource changed: %q, %v", rebuiltArchive, err)
	}
	entries, err := pLST(output+".lst", key, lstMoon)
	if err != nil {
		t.Fatal(err)
	}
	if got := entries[0].FN(); got != normalizedFilename {
		t.Fatalf("expected rebuilt name %q, got %q", normalizedFilename, got)
	}
	rebuiltIndex, err := os.ReadFile(output + ".lst")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rebuiltIndex[12:12+moonFnSz], lstData[12:12+moonFnSz]) {
		t.Fatal("MOON patch changed the original filename bytes")
	}
}

func TestMoonPatchRejectsInPlaceOutput(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "moon")
	if err := os.WriteFile(archive, []byte("original archive"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive+".lst", []byte("original index"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cmdPatch(archive, root, archive, -1, -1); err == nil {
		t.Fatal("expected in-place patch to be rejected")
	}
	for path, want := range map[string]string{archive: "original archive", archive + ".lst": "original index"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("in-place rejection modified %s: %q, %v", path, got, err)
		}
	}
}

func TestParseAndRenderOldMoonSNX(t *testing.T) {
	data := []byte{
		0x00, 0x0F, 0x00, 0x01, 0x00, // X_00 000F 0001
		0x40, 'D', 'a', 'y', ' ', '1', 0x00,
		0x15, 0x01, 0x02, 'H', 'e', 'l', 'l', 'o', 0x81, 0x40, 'M', 'O', 'O', 'N', 0x05, 0x00,
		0x2C, 0x01,
		0x17, 0x03, 0x00, 'I', 't', '\'', 's', 0x81, 0x40, 'f', 'i', 'n', 'e', 0x00, 0x00,
		0xFF,
	}
	instructions, err := parseMoonOldSNX(data)
	if err != nil {
		t.Fatal(err)
	}
	text, err := renderMoonOldSNX("TEST.snx", instructions)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"X_00 000F 0001",
		"SETSTATUS 'Day 1'",
		"TEXT Hello MOON",
		"STOP_SOUND 01",
		"SELECT 0003 'It\\'s fine'",
		"EOF",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered script missing %q:\n%s", want, text)
		}
	}
}

func TestOldMoonSNXTextLineBreakIsStable(t *testing.T) {
	data := []byte{0x15, 0x01, 0x02, 'A', ' ', ' ', 0x01, 'B', 0x05, 0x00, 0xFF}
	instructions, err := parseMoonOldSNX(data)
	if err != nil {
		t.Fatal(err)
	}
	text, err := renderMoonOldSNX("TEST.snx", instructions)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(text), []byte("TEXT A\\nB")) {
		t.Fatalf("expected normalized line break, got %q", text)
	}
}

func TestApplyMoonAccentSentinelsOnlyInsideStrings(t *testing.T) {
	// X_24 has an operand equal to the first sentinel. TEXT and SELECT contain
	// sentinel bytes which must become hook accent bytes. Structural bytes must
	// remain bit-for-bit identical.
	input := []byte{
		0x24, 0x02,
		0x15, 0x01, 0x02, 'C', 'a', 'f', 0x02, 0x05, 0x00,
		0x17, 0x00, 0x00, 't', 'r', 0x03, 's', 0x00, 0x00,
		0xFF,
	}
	got, count, err := applyMoonAccentSentinels(input)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 replacements, got %d", count)
	}
	if got[1] != 0x02 || got[3] != 0x01 || got[4] != 0x02 {
		t.Fatalf("structural bytes changed: % X", got[:5])
	}
	if got[8] != 0xA1 || got[16] != 0xA2 {
		t.Fatalf("string sentinels were not replaced: % X", got)
	}
}

func TestOldMoonSNXAcceptsOrphanedBytesAfterEOF(t *testing.T) {
	instructions, err := parseMoonOldSNX([]byte{0x15, 0x01, 0x02, 'A', 0x05, 0x00, 0xFF, 0x15, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	if len(instructions) != 2 || instructions[1].opcode != 0xFF || len(instructions[1].bytes) != 2 {
		t.Fatalf("unexpected parsed instructions: %#v", instructions)
	}
}
