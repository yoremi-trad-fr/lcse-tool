package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHookNativeInstallIntegration(t *testing.T) {
	source, kit := os.Getenv("LCSE_TEST_HOOK_ENGINE"), os.Getenv("LCSE_TEST_HOOK_KIT")
	if source == "" || kit == "" {
		t.Skip("set original engine and built 1.4 hook kit")
	}
	engine, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	game := t.TempDir()
	if err := os.WriteFile(filepath.Join(game, "MOON_eng.EXE"), engine, 0644); err != nil {
		t.Fatal(err)
	}
	writeTestMoonLocale(t, game)
	old := []byte("previous launcher retained in backup")
	for _, name := range []string{"MOON_fr.exe", "MOON_fr_hook.exe", "MOON.bat", "MOON_FR.bat"} {
		if err := os.WriteFile(filepath.Join(game, name), old, 0644); err != nil {
			t.Fatal(err)
		}
	}
	app := NewApp()
	app.oneHookDir = kit
	if got := app.InstallMoonHook(game, "MS Gothic", "0"); got != "OK" {
		t.Fatal(got)
	}
	patched, err := os.ReadFile(filepath.Join(game, "MOON_eng.EXE"))
	if err != nil {
		t.Fatal(err)
	}
	before, err := pe.NewFile(bytes.NewReader(engine))
	if err != nil {
		t.Fatal(err)
	}
	defer before.Close()
	after, err := pe.NewFile(bytes.NewReader(patched))
	if err != nil {
		t.Fatal(err)
	}
	defer after.Close()
	if before.OptionalHeader.(*pe.OptionalHeader32).AddressOfEntryPoint != after.OptionalHeader.(*pe.OptionalHeader32).AddressOfEntryPoint {
		t.Fatal("entry point changed")
	}
	for _, s := range before.Sections {
		if s.Name == ".text" {
			data, _ := s.Data()
			other, _ := after.Section(".text").Data()
			if !bytes.Equal(data, other) {
				t.Fatal("game instructions changed")
			}
		}
	}
	found := false
	// debug/pe.ImportedSymbols assumes DLL names share the import section;
	// this copy deliberately stores its DLL name in a separate data section.
	h := after.OptionalHeader.(*pe.OptionalHeader32)
	for i := uint32(0); i < 100; i++ {
		o, err := peRawOffset(after, h.DataDirectory[1].VirtualAddress+i*20, 20, patched)
		if err != nil {
			t.Fatal(err)
		}
		rva := binary.LittleEndian.Uint32(patched[o+12:])
		if rva == 0 {
			break
		}
		name, err := peString(after, rva, patched)
		if err != nil {
			t.Fatal(err)
		}
		if name == "lcse_hook.dll" {
			found = true
		}
	}
	if !found {
		t.Fatal("normal GDI import missing")
	}
	original, manifest, err := readMoonOriginal(game)
	if err != nil || manifest == nil || !bytes.Equal(original, engine) {
		t.Fatal("original engine backup invalid", err)
	}
	for _, name := range []string{"MOON_fr.exe", "MOON_fr_hook.exe", "MOON.bat"} {
		if _, err := os.Stat(filepath.Join(game, name)); !os.IsNotExist(err) {
			t.Fatal("obsolete launcher still present", name)
		}
	}
	bat, err := os.ReadFile(filepath.Join(game, "MOON_FR.bat"))
	if err != nil || !bytes.Contains(bat, []byte(`-runas JAP "%~dp0MOON_eng.EXE"`)) || bytes.Contains(bat, []byte("MOON_fr.exe")) {
		t.Fatal("canonical Japanese-locale launcher missing", err)
	}
	backups, _ := filepath.Glob(filepath.Join(game, "lcse_backup_*", "MOON_fr.exe"))
	if len(backups) != 1 {
		t.Fatal("previous launcher backup missing")
	}
	saved, _ := os.ReadFile(backups[0])
	if !bytes.Equal(saved, old) {
		t.Fatal("backup altered")
	}
	if got := app.InstallMoonHook(game, "MS Gothic", "0"); got != "OK" {
		t.Fatal("reinstall failed")
	}
	reinstalled, _ := os.ReadFile(filepath.Join(game, "MOON_eng.EXE"))
	if !bytes.Equal(reinstalled, patched) {
		t.Fatal("reinstall changed engine again")
	}
	if got := app.InstallMoonHook(game, "MS Gothic\nLog=1", "0"); got != "ERROR" {
		t.Fatal("multiline font accepted")
	}
	dll, err := os.ReadFile(filepath.Join(kit, "lcse_hook.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildAccentedGame(patched, dll); err == nil {
		t.Fatal("already patched engine accepted")
	}
	if _, err := buildAccentedGame(engine, []byte("legacy DLL")); err == nil {
		t.Fatal("invalid hook DLL accepted")
	}
	// Reject a damaged or escaped original backup without changing outputs.
	originalPath := manifest.OriginalPath
	for _, path := range []string{"../outside.EXE", filepath.Join(t.TempDir(), "outside.EXE"), ""} {
		manifest.OriginalPath = path
		data, _ := json.Marshal(manifest)
		os.WriteFile(filepath.Join(game, moonHookManifestName), data, 0644)
		if app.InstallMoonHook(game, "MS Gothic", "0") != "ERROR" {
			t.Fatal("unsafe backup accepted", path)
		}
	}
	manifest.OriginalPath = originalPath
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(game, moonHookManifestName), data, 0644)
	os.WriteFile(filepath.Join(game, originalPath), []byte("damaged"), 0644)
	if app.InstallMoonHook(game, "MS Gothic", "0") != "ERROR" {
		t.Fatal("damaged original accepted")
	}
	os.WriteFile(filepath.Join(game, originalPath), engine, 0644)
	modified := append([]byte(nil), patched...)
	modified[len(modified)-1] ^= 1
	os.WriteFile(filepath.Join(game, "MOON_eng.EXE"), modified, 0644)
	if app.InstallMoonHook(game, "MS Gothic", "0") != "ERROR" {
		t.Fatal("externally changed engine accepted")
	}
	afterFailure, _ := os.ReadFile(filepath.Join(game, "MOON_eng.EXE"))
	if !bytes.Equal(afterFailure, modified) {
		t.Fatal("failed preflight modified engine")
	}
	t.Log("Japanese BAT, normal imports, unchanged machine code/entry point, verified original backups and safe reinstall")
}

func writeTestMoonLocale(t *testing.T, game string) {
	t.Helper()
	dir := filepath.Join(game, "locale")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LEProc.exe", "LECommonLibrary.dll", "LoaderDll.dll", "LocaleEmulator.dll"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test fixture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	config := `<LEConfig><Profiles><Profile Guid="JAP"><Location>ja-JP</Location><RunAsAdmin>false</RunAsAdmin><RunWithSuspend>false</RunWithSuspend></Profile></Profiles></LEConfig>`
	if err := os.WriteFile(filepath.Join(dir, "LEConfig.xml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestMoonLocalePreflight(t *testing.T) {
	game := t.TempDir()
	writeTestMoonLocale(t, game)
	if err := checkMoonLocale(game); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(game, "locale", "LEConfig.xml")
	valid, _ := os.ReadFile(config)
	for _, pair := range [][2]string{{"ja-JP", "fr-FR"}, {"<RunAsAdmin>false", "<RunAsAdmin>true"}, {"<RunWithSuspend>false", "<RunWithSuspend>true"}, {`Guid="JAP"`, `Guid="JAPADMIN"`}} {
		os.WriteFile(config, []byte(strings.ReplaceAll(string(valid), pair[0], pair[1])), 0644)
		if err := checkMoonLocale(game); err == nil {
			t.Fatal("invalid locale accepted", pair)
		}
	}
	os.WriteFile(config, valid, 0644)
	os.Remove(filepath.Join(game, "locale", "LECommonLibrary.dll"))
	if err := checkMoonLocale(game); err == nil {
		t.Fatal("missing locale dependency accepted")
	}
}

func TestHookRejectsInvalidPE(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("not an executable"), bytes.Repeat([]byte{0}, 1024)} {
		if _, err := buildAccentedGame(data, nil); err == nil {
			t.Fatal("invalid engine accepted")
		}
		if _, err := hookExports(data); err == nil {
			t.Fatal("invalid DLL accepted")
		}
	}
}

func TestHookInstallPreflightPreservesExistingFiles(t *testing.T) {
	game := t.TempDir()
	original := []byte("keep")
	if err := os.WriteFile(filepath.Join(game, "MOON_fr.exe"), original, 0644); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.oneHookDir = t.TempDir()
	if app.InstallMoonHook(game, "MS Gothic", "0") != "ERROR" {
		t.Fatal("missing engine accepted")
	}
	after, _ := os.ReadFile(filepath.Join(game, "MOON_fr.exe"))
	if !bytes.Equal(original, after) {
		t.Fatal("preflight modified output")
	}
}
