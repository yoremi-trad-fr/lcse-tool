package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const moonHookManifestName = "lcse_hook_install.json"

type moonHookManifest struct {
	Version         int    `json:"version"`
	OriginalPath    string `json:"originalPath"`
	OriginalSHA256  string `json:"originalSHA256"`
	InstalledSHA256 string `json:"installedSHA256"`
	DLLSHA256       string `json:"dllSHA256"`
}

func hookHash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func readMoonOriginal(gameDir string) ([]byte, *moonHookManifest, error) {
	current, err := os.ReadFile(filepath.Join(gameDir, "MOON_eng.EXE"))
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(filepath.Join(gameDir, moonHookManifestName))
	if os.IsNotExist(err) {
		return current, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var m moonHookManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, nil, fmt.Errorf("manifeste du hook invalide : %w", err)
	}
	if m.Version != 1 || filepath.IsAbs(m.OriginalPath) || m.OriginalPath == "" {
		return nil, nil, fmt.Errorf("sauvegarde du moteur invalide")
	}
	root, err := filepath.Abs(gameDir)
	if err != nil {
		return nil, nil, err
	}
	path := filepath.Join(root, m.OriginalPath)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, nil, fmt.Errorf("sauvegarde du moteur hors du dossier du jeu")
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("sauvegarde originale introuvable : %w", err)
	}
	if hookHash(original) != m.OriginalSHA256 {
		return nil, nil, fmt.Errorf("empreinte du moteur sauvegarde incorrecte")
	}
	if hash := hookHash(current); hash != m.InstalledSHA256 && hash != m.OriginalSHA256 {
		return nil, nil, fmt.Errorf("le moteur a change depuis l'installation ; restaurer l'original avant de reinstaller le hook")
	}
	return original, &m, nil
}

func checkMoonLocale(gameDir string) error {
	for _, name := range []string{"LEProc.exe", "LECommonLibrary.dll", "LoaderDll.dll", "LocaleEmulator.dll"} {
		if !fileExists(filepath.Join(gameDir, "locale", name)) {
			return fmt.Errorf("locale/%s introuvable : conserver le dossier locale du jeu", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(gameDir, "locale", "LEConfig.xml"))
	if err != nil {
		return err
	}
	var config struct {
		Profiles []struct {
			Guid           string `xml:"Guid,attr"`
			Location       string `xml:"Location"`
			RunAsAdmin     bool   `xml:"RunAsAdmin"`
			RunWithSuspend bool   `xml:"RunWithSuspend"`
		} `xml:"Profiles>Profile"`
	}
	if err := xml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("LEConfig.xml invalide : %w", err)
	}
	for _, p := range config.Profiles {
		if p.Guid == "JAP" && p.Location == "ja-JP" && !p.RunAsAdmin && !p.RunWithSuspend {
			return nil
		}
	}
	return fmt.Errorf("profil JAP japonais sans elevation requis dans locale/LEConfig.xml")
}

// MOON/Locale Emulator resolves the archive and working folder from the
// executable's real path. Install the GDI import in place after saving the
// original executable; its original name and the archive paths stay intact.
// The sole entry point is MOON_FR.bat -> LEProc -runas JAP MOON_eng.EXE.
func (a *App) installMoonLocaleHook(gameDir, fontName, debugLog string) string {
	fail := func(err error) string { a.logError(err.Error()); return "ERROR" }
	if gameDir == "" {
		return fail(fmt.Errorf("dossier du jeu MOON requis"))
	}
	if err := checkMoonLocale(gameDir); err != nil {
		return fail(err)
	}
	if strings.TrimSpace(fontName) == "" {
		fontName = "MS Gothic"
	}
	if debugLog == "" {
		debugLog = "0"
	}
	if strings.ContainsAny(fontName, "\r\n") || len(fontName) > 31 || (debugLog != "0" && debugLog != "1" && debugLog != "2") {
		return fail(fmt.Errorf("police ou niveau de journal invalide"))
	}
	if a.oneHookDir == "" {
		a.oneHookDir = a.findDir([]string{"one_hook", "Hook_v5.1"})
	}
	original, manifest, err := readMoonOriginal(gameDir)
	if err != nil {
		return fail(err)
	}
	dll, err := os.ReadFile(filepath.Join(a.oneHookDir, "lcse_hook.dll"))
	if err != nil {
		return fail(err)
	}
	patched, err := buildAccentedGame(original, dll)
	if err != nil {
		return fail(err)
	}
	bat, err := os.ReadFile(filepath.Join(a.oneHookDir, "MOON_FR.bat"))
	if err != nil {
		return fail(err)
	}
	backupName := "lcse_backup_" + time.Now().Format("20060102_150405.000000000")
	backup := filepath.Join(gameDir, backupName)
	if manifest == nil {
		manifest = &moonHookManifest{Version: 1, OriginalPath: filepath.Join(backupName, "MOON_eng.EXE"), OriginalSHA256: hookHash(original)}
	}
	manifest.InstalledSHA256 = hookHash(patched)
	manifest.DLLSHA256 = hookHash(dll)
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fail(err)
	}
	manifestData = append(manifestData, '\n')
	files := map[string][]byte{"MOON_eng.EXE": patched, "lcse_hook.dll": dll, "lcse_hook.ini": []byte("[Font]\r\nName=" + fontName + "\r\n\r\n[Debug]\r\nLog=" + debugLog + "\r\n"), "MOON_FR.bat": bat, moonHookManifestName: manifestData,
		"MOON.bat": nil, "MOON_fr.exe": nil, "MOON_fr_hook.exe": nil}
	order := []string{"lcse_hook.dll", "lcse_hook.ini", "MOON_eng.EXE", "MOON_FR.bat", moonHookManifestName, "MOON.bat", "MOON_fr.exe", "MOON_fr_hook.exe"}
	old := map[string][]byte{}
	for name := range files {
		data, err := os.ReadFile(filepath.Join(gameDir, name))
		if err == nil {
			old[name] = data
		} else if !os.IsNotExist(err) {
			return fail(err)
		}
	}
	if err := os.Mkdir(backup, 0755); err != nil {
		return fail(err)
	}
	for name, data := range old {
		if err := os.WriteFile(filepath.Join(backup, name), data, 0644); err != nil {
			return fail(err)
		}
	}
	// Backups finish before the first replacement. On a failed replacement,
	// restore every previous file, including the original executable.
	installed := []string{}
	rollback := func(cause error) string {
		for _, name := range installed {
			var err error
			if previous, ok := old[name]; ok {
				err = os.WriteFile(filepath.Join(gameDir, name), previous, 0644)
			} else {
				err = os.Remove(filepath.Join(gameDir, name))
				if os.IsNotExist(err) {
					err = nil
				}
			}
			if err != nil {
				a.logError("Restauration : " + err.Error())
			}
		}
		return fail(cause)
	}
	for _, name := range order {
		data := files[name]
		path := filepath.Join(gameDir, name)
		if data == nil {
			if _, ok := old[name]; !ok {
				continue
			}
		}
		installed = append(installed, name)
		if data == nil {
			err = os.Remove(path)
		} else {
			err = os.WriteFile(path, data, 0644)
		}
		if err != nil {
			return rollback(err)
		}
	}
	a.log("Sauvegarde : " + backup)
	a.logOK("Hook MOON installe avec locale japonaise. Lancer uniquement MOON_FR.bat.")
	return "OK"
}
