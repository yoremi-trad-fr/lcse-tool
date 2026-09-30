package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMoonBundledScriptsToUTF(t *testing.T) {
	app := NewApp()
	sources := t.TempDir()
	if err := os.WriteFile(filepath.Join(sources, "INIT.txt"), []byte("TEXT test\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	app.moonScripts = sources
	out := t.TempDir()

	result := app.MoonBundledScriptsToUTF(out)
	if result.Status != "OK" {
		t.Fatalf("expected OK, got %s: %s", result.Status, result.Detail)
	}

	files, err := txtFiles(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 script, got %d", len(files))
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

func TestMoonActiveTxtFilesSkipsLegacySplitScripts(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"DAY01.txt", "DAY01A.txt", "DAY01B.txt", "DAY02.txt", "DAY02A.txt", "INIT.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("EOF\r\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := moonActiveTxtFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, file := range files {
		got[filepath.Base(file)] = true
	}
	if got["DAY01.txt"] {
		t.Fatal("DAY01.txt must be skipped when both split replacements exist")
	}
	if !got["DAY02.txt"] {
		t.Fatal("DAY02.txt must remain when a split replacement is missing")
	}
	for _, want := range []string{"DAY01A.txt", "DAY01B.txt", "DAY02A.txt", "INIT.txt"} {
		if !got[want] {
			t.Fatalf("expected active script %s", want)
		}
	}
}

func TestResolveMoonScriptsNextToExtractedFolder(t *testing.T) {
	app := NewApp()
	root := t.TempDir()
	extracted := filepath.Join(root, "Extract")
	sources := filepath.Join(root, "moon_scripts")
	if err := os.MkdirAll(extracted, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sources, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sources, "INIT.txt"), []byte("TEXT test\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := app.resolveMoonScripts(extracted); got != sources {
		t.Fatalf("expected %s, got %s", sources, got)
	}
}

func TestMoonCellEscapeRoundTrip(t *testing.T) {
	original := "literal \\n and \\t and \\\\ plus\ta tab\nand a newline"
	if got := unescapeMoonCell(escapeMoonCell(original)); got != original {
		t.Fatalf("round trip mismatch:\nwant %q\n got %q", original, got)
	}
}

func TestMoonImportEscapesStatusApostrophe(t *testing.T) {
	scripts := t.TempDir()
	dialogues := t.TempDir()
	out := t.TempDir()
	script := filepath.Join(scripts, "TEST.txt")
	dialogue := filepath.Join(dialogues, "TEST.dlg.txt")
	if err := writeUTF8BOM(script, "SETSTATUS 'Original' \r\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeUTF8BOM(dialogue, "id\tline\ttag\ttext\r\n1\t1\tSETSTATUS\tL'histoire\r\n"); err != nil {
		t.Fatal(err)
	}

	result := NewApp().MoonImportDialogues(scripts, dialogues, out)
	if result.Status != "OK" {
		t.Fatalf("expected OK, got %s: %s", result.Status, result.Detail)
	}
	text, err := readTextAuto(filepath.Join(out, "TEST.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "SETSTATUS 'L\\'histoire'") {
		t.Fatalf("escaped apostrophe not found: %q", text)
	}
}

func TestMoonDialogueRoundTripPreservesStatusSuffix(t *testing.T) {
	scripts := t.TempDir()
	dialogues := t.TempDir()
	out := t.TempDir()
	source := "SETSTATUS 'L\\'histoire' \t\r\n"
	if err := writeUTF8BOM(filepath.Join(scripts, "TEST.txt"), source); err != nil {
		t.Fatal(err)
	}
	if _, err := exportMoonDialogueFile(filepath.Join(scripts, "TEST.txt"), dialogues); err != nil {
		t.Fatal(err)
	}
	if _, err := importMoonDialogueFile(filepath.Join(scripts, "TEST.txt"), filepath.Join(dialogues, "TEST.dlg.txt"), filepath.Join(out, "TEST.txt")); err != nil {
		t.Fatal(err)
	}
	got, err := readTextAuto(filepath.Join(out, "TEST.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("status line changed during dialogue round trip:\nwant %q\n got %q", source, got)
	}
}

func TestInstallMoonHookCreatesFrenchLauncherWithoutRenamingEngine(t *testing.T) {
	kit := t.TempDir()
	game := t.TempDir()
	for _, name := range []string{"moon_launcher.exe", "lcse_hook.dll", "lcse_hook.ini"} {
		if err := os.WriteFile(filepath.Join(kit, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	engine := []byte("original MOON engine")
	if err := os.WriteFile(filepath.Join(game, "MOON_eng.EXE"), engine, 0644); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.oneHookDir = kit
	if got := app.InstallMoonHook(game, "MS Gothic", "0"); got != "OK" {
		t.Fatalf("expected OK, got %s", got)
	}
	if !fileExists(filepath.Join(game, "MOON_fr.exe")) {
		t.Fatal("MOON_fr.exe launcher was not installed")
	}
	gotEngine, err := os.ReadFile(filepath.Join(game, "MOON_eng.EXE"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotEngine, engine) {
		t.Fatal("MOON_eng.EXE was modified")
	}
}

func TestEncodeMoonAssemblerSourceUsesSingleByteAccentSentinels(t *testing.T) {
	encoded, err := encodeMoonAssemblerSource("Café, où êtes-vous ?\\nSuite")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []byte{0x02, 0x11, 0x0F} {
		if !bytes.Contains(encoded, []byte{want}) {
			t.Fatalf("sentinel %02X missing from % X", want, encoded)
		}
	}
	if bytes.Contains(encoded, []byte{0xA1}) {
		t.Fatalf("final hook byte must not be written before assembly: % X", encoded)
	}
	if bytes.Contains(encoded, []byte{'\\', 'n'}) {
		t.Fatalf("soft SNX line break must be left to moon_asm: % X", encoded)
	}
}

func TestEncodeMoonAssemblerSourceEscapesShiftJISTrailBackslash(t *testing.T) {
	encoded, err := encodeMoonAssemblerSource("TEXT 構\r\nSELECT 0001 '―'\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte{0x8D, 0x5C, 0x5C}) {
		t.Fatalf("TEXT trail byte was not escaped: % X", encoded)
	}
	if bytes.Contains(encoded, []byte{0x81, 0x5C, 0x5C}) || !bytes.Contains(encoded, []byte{0x81, 0x5C}) {
		t.Fatalf("quoted SELECT trail byte must remain unchanged: % X", encoded)
	}
}

func TestMoonTXTToSNXIntegration(t *testing.T) {
	moonAsm := os.Getenv("LCSE_TEST_MOON_ASM")
	lcseTool := os.Getenv("LCSE_TEST_TOOL")
	if moonAsm == "" || lcseTool == "" {
		t.Skip("set LCSE_TEST_MOON_ASM and LCSE_TEST_TOOL for the external assembler test")
	}
	tmp := t.TempDir()
	source := filepath.Join(tmp, "ACCENT.txt")
	output := filepath.Join(tmp, "ACCENT.snx")
	if err := writeUTF8BOM(source, "# MOON SCRIPT FILE \"ACCENT.SNX\"\r\n\r\nTEXT Café où êtes-vous ?\r\nEOF\r\n"); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.moonAsm = moonAsm
	app.lcsePath = lcseTool
	if got := app.MoonTXTToSNX(source, output); got != "OK" {
		t.Fatalf("expected OK, got %s", got)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []byte{0xA1, 0xAA, 0xA8} {
		if !bytes.Contains(data, []byte{want}) {
			t.Fatalf("final accent byte %02X missing from % X", want, data)
		}
	}
}

func TestMoonAssembleScriptsIntegration(t *testing.T) {
	moonAsm := os.Getenv("LCSE_TEST_MOON_ASM")
	lcseTool := os.Getenv("LCSE_TEST_TOOL")
	scripts := os.Getenv("LCSE_TEST_MOON_SCRIPTS")
	if moonAsm == "" || lcseTool == "" || scripts == "" {
		t.Skip("set LCSE_TEST_MOON_ASM, LCSE_TEST_TOOL and LCSE_TEST_MOON_SCRIPTS for the batch test")
	}
	tmp := os.Getenv("LCSE_TEST_MOON_OUTPUT")
	if tmp == "" {
		tmp = t.TempDir()
	} else if err := os.MkdirAll(tmp, 0755); err != nil {
		t.Fatal(err)
	}
	built := filepath.Join(tmp, "built")
	roundTrip := filepath.Join(tmp, "roundtrip")
	app := NewApp()
	app.moonAsm = moonAsm
	app.lcsePath = lcseTool
	if got := app.MoonAssembleScripts(scripts, built); got != "OK" {
		t.Fatalf("expected OK, got %s", got)
	}
	snxFiles, err := filepath.Glob(filepath.Join(built, "*.[sS][nN][xX]"))
	if err != nil {
		t.Fatal(err)
	}
	active, err := moonActiveTxtFiles(scripts)
	if err != nil {
		t.Fatal(err)
	}
	if len(snxFiles) != len(active) {
		t.Fatalf("expected %d active SNX, got %d", len(active), len(snxFiles))
	}
	if err := app.runLCSE("snx2txt", built, roundTrip); err != nil {
		t.Fatal(err)
	}
	for _, source := range active {
		want, err := readTextAuto(source)
		if err != nil {
			t.Fatal(err)
		}
		got, err := readTextAuto(filepath.Join(roundTrip, filepath.Base(source)))
		if err != nil {
			t.Fatal(err)
		}
		want = normalizeMoonSourceForRoundTripTest(want)
		got = normalizeMoonSourceForRoundTripTest(got)
		if got != want {
			wantLines := strings.Split(want, "\r\n")
			gotLines := strings.Split(got, "\r\n")
			for line := 0; line < len(wantLines) && line < len(gotLines); line++ {
				if wantLines[line] != gotLines[line] {
					t.Fatalf("round-trip source mismatch: %s line %d\nwant %q\n got %q", filepath.Base(source), line+1, wantLines[line], gotLines[line])
				}
			}
			t.Fatalf("round-trip source mismatch: %s (%d vs %d lines)", filepath.Base(source), len(wantLines), len(gotLines))
		}
	}
}

func normalizeMoonSourceForRoundTripTest(text string) string {
	lines := strings.Split(strings.ReplaceAll(normalizeNewlines(text), "\r\n", "\n"), "\n")
	commands := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if line == "" || strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			continue
		}
		commands = append(commands, line)
	}
	text = normalizeMoonSoftBreaksForTest(strings.Join(commands, "\r\n"))
	lines = strings.Split(text, "\r\n")
	for index := range lines {
		lines[index] = strings.TrimRight(lines[index], " \t")
		if lines[index] == "TEXT ^" {
			lines[index] = "TEXT"
		}
	}
	text = strings.Join(lines, "\r\n")
	labelPattern := regexp.MustCompile(`[\.:]LABEL[0-9]+`)
	labels := map[string]string{}
	return labelPattern.ReplaceAllStringFunc(text, func(label string) string {
		key := label[1:]
		if normalized, ok := labels[key]; ok {
			return label[:1] + normalized
		}
		normalized := "L" + string(rune('A'+len(labels)))
		labels[key] = normalized
		return label[:1] + normalized
	})
}

func TestMoonDialoguePipelineIntegration(t *testing.T) {
	scripts := os.Getenv("LCSE_TEST_MOON_SCRIPTS")
	if scripts == "" {
		t.Skip("set LCSE_TEST_MOON_SCRIPTS for the dialogue pipeline test")
	}
	tmp := t.TempDir()
	dialogues := os.Getenv("LCSE_TEST_MOON_DIALOGUES_OUTPUT")
	if dialogues == "" {
		dialogues = filepath.Join(tmp, "dialogues")
	} else if err := os.MkdirAll(dialogues, 0755); err != nil {
		t.Fatal(err)
	}
	imported := filepath.Join(tmp, "imported")
	app := NewApp()
	exported := app.MoonExportDialogues(scripts, dialogues)
	if exported.Status != "OK" {
		t.Fatalf("export failed: %s", exported.Detail)
	}
	t.Log(exported.Detail)
	result := app.MoonImportDialogues(scripts, dialogues, imported)
	if result.Status != "OK" {
		t.Fatalf("import failed: %s", result.Detail)
	}
	t.Log(result.Detail)
	files, err := txtFiles(imported)
	if err != nil {
		t.Fatal(err)
	}
	active, err := moonActiveTxtFiles(scripts)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(active) {
		t.Fatalf("expected %d complete imported scripts, got %d", len(active), len(files))
	}
	for _, gotFile := range files {
		wantFile := filepath.Join(scripts, filepath.Base(gotFile))
		want, err := readTextAuto(wantFile)
		if err != nil {
			t.Fatal(err)
		}
		got, err := readTextAuto(gotFile)
		if err != nil {
			t.Fatal(err)
		}
		want = normalizeNewlines(want)
		got = normalizeNewlines(got)
		if got != want {
			wantLines := strings.Split(want, "\r\n")
			gotLines := strings.Split(got, "\r\n")
			for line := 0; line < len(wantLines) && line < len(gotLines); line++ {
				if wantLines[line] != gotLines[line] {
					t.Fatalf("unchanged dialogue round trip altered %s line %d\nwant %q\n got %q", filepath.Base(gotFile), line+1, wantLines[line], gotLines[line])
				}
			}
			t.Fatalf("unchanged dialogue round trip altered %s", filepath.Base(gotFile))
		}
	}
}

func TestMoonScriptsToUTFIntegration(t *testing.T) {
	source := os.Getenv("LCSE_TEST_MOON_SCRIPTS_SOURCE")
	output := os.Getenv("LCSE_TEST_MOON_UTF_OUTPUT")
	if source == "" || output == "" {
		t.Skip("set LCSE_TEST_MOON_SCRIPTS_SOURCE and LCSE_TEST_MOON_UTF_OUTPUT for the UTF-8 conversion test")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	result := app.MoonScriptsToUTF(source, output)
	if result.Status != "OK" {
		t.Fatalf("conversion failed: %s", result.Detail)
	}
	sourceFiles, err := txtFiles(source)
	if err != nil {
		t.Fatal(err)
	}
	outputFiles, err := txtFiles(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(outputFiles) != len(sourceFiles) {
		t.Fatalf("expected %d UTF-8 scripts, got %d", len(sourceFiles), len(outputFiles))
	}
	for _, outputFile := range outputFiles {
		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 3 || data[0] != 0xEF || data[1] != 0xBB || data[2] != 0xBF {
			t.Fatalf("expected UTF-8 BOM in %s", outputFile)
		}
	}
}

func TestMoonTranslatedDialogueBuildIntegration(t *testing.T) {
	script := os.Getenv("LCSE_TEST_MOON_TRANSLATED_SCRIPT")
	dialogue := os.Getenv("LCSE_TEST_MOON_TRANSLATED_DIALOGUE")
	moonAsm := os.Getenv("LCSE_TEST_MOON_ASM")
	lcseTool := os.Getenv("LCSE_TEST_TOOL")
	if script == "" || dialogue == "" || moonAsm == "" || lcseTool == "" {
		t.Skip("set LCSE_TEST_MOON_TRANSLATED_SCRIPT, LCSE_TEST_MOON_TRANSLATED_DIALOGUE, LCSE_TEST_MOON_ASM and LCSE_TEST_TOOL")
	}
	tmp := t.TempDir()
	translatedScript := filepath.Join(tmp, filepath.Base(script))
	count, err := importMoonDialogueFile(script, dialogue, translatedScript)
	if err != nil {
		t.Fatal(err)
	}
	dialogues, err := readMoonDialogueFile(dialogue)
	if err != nil {
		t.Fatal(err)
	}
	if count != len(dialogues) {
		t.Fatalf("expected %d imported entries, got %d", len(dialogues), count)
	}
	app := NewApp()
	app.moonAsm = moonAsm
	app.lcsePath = lcseTool
	outputSNX := filepath.Join(tmp, strings.TrimSuffix(filepath.Base(script), filepath.Ext(script))+".SNX")
	if got := app.MoonTXTToSNX(translatedScript, outputSNX); got != "OK" {
		t.Fatalf("expected translated SNX build to succeed, got %s", got)
	}
	info, err := os.Stat(outputSNX)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() >= 64*1024 {
		t.Fatalf("translated SNX is too large for the engine: %d bytes", info.Size())
	}
}

func normalizeMoonSoftBreaksForTest(text string) string {
	values := []rune(text)
	var out strings.Builder
	for index := 0; index < len(values); index++ {
		if values[index] == '\\' && index+1 < len(values) && values[index+1] == 'n' {
			if index > 0 && index+2 < len(values) && values[index-1] <= 0xFF && values[index+2] <= 0xFF {
				out.WriteByte(' ')
			}
			index++
			continue
		}
		out.WriteRune(values[index])
	}
	normalized := out.String()
	normalized = strings.ReplaceAll(normalized, "\x01", "")
	normalized = strings.ReplaceAll(normalized, "\u3000", " ")
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}
	return normalized
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
