package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoonBundledScriptsToUTF(t *testing.T) {
	app := NewApp()
	out := t.TempDir()

	result := app.MoonBundledScriptsToUTF(out)
	if result.Status != "OK" {
		t.Fatalf("expected OK, got %s: %s", result.Status, result.Detail)
	}

	files, err := txtFiles(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 125 {
		t.Fatalf("expected 125 scripts, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 3 || data[0] != 0xEF || data[1] != 0xBB || data[2] != 0xBF {
		t.Fatalf("expected UTF-8 BOM in %s", files[0])
	}
}

func TestMoonScriptsToUTFRejectsSNXOnlyFolder(t *testing.T) {
	app := NewApp()
	input := t.TempDir()
	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(input, "A_D2.snx"), []byte{0x00, 0x01}, 0644); err != nil {
		t.Fatal(err)
	}

	result := app.MoonScriptsToUTF(input, output)
	if result.Status != "ERROR" {
		t.Fatalf("expected ERROR, got %s: %s", result.Status, result.Detail)
	}
}

func TestMoonSNXToTXTUsesBundledSource(t *testing.T) {
	app := NewApp()
	input := t.TempDir()
	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(input, "A_D2.snx"), []byte{0x00, 0x01}, 0644); err != nil {
		t.Fatal(err)
	}

	if status := app.MoonSNXToTXT(input, output); status != "OK" {
		t.Fatalf("expected OK, got %s", status)
	}
	if !fileExists(filepath.Join(output, "A_D2.txt")) {
		t.Fatalf("expected A_D2.txt")
	}
}

func TestOneDialogueExportImport(t *testing.T) {
	app := NewApp()
	scripts := t.TempDir()
	dialogues := t.TempDir()
	patched := t.TempDir()
	source := "0\t0x0000\tRES\tBG001\r\n1\t0x0004\tTXT\tBonjour\r\n2\t0x0008\tTXT\tAu revoir\r\n"
	if err := writeUTF8BOM(filepath.Join(scripts, "TEST.txt"), source); err != nil {
		t.Fatal(err)
	}

	result := app.OneExportDialogues(scripts, dialogues)
	if result.Status != "OK" {
		t.Fatalf("expected export OK, got %s: %s", result.Status, result.Detail)
	}
	dialogueFile := filepath.Join(dialogues, "TEST.dlg.txt")
	text, err := readTextAuto(dialogueFile)
	if err != nil {
		t.Fatal(err)
	}
	text = strings.Replace(text, "Bonjour", "Salut", 1)
	if err := writeUTF8BOM(dialogueFile, text); err != nil {
		t.Fatal(err)
	}

	result = app.OneImportDialogues(scripts, dialogues, patched)
	if result.Status != "OK" {
		t.Fatalf("expected import OK, got %s: %s", result.Status, result.Detail)
	}
	out, err := readTextAuto(filepath.Join(patched, "TEST.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\tTXT\tSalut") {
		t.Fatalf("patched text not found: %s", out)
	}
}
