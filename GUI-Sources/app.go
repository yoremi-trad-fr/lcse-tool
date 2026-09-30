package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	lcsePath    string
	moonAsm     string
	moonTGF     string
	moonScripts string
	oneHookDir  string
	cancelFunc  context.CancelFunc
	mu          sync.Mutex
}

type BatchResult struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.findLCSETool()
	a.moonAsm = a.findTool([]string{"moon_asm.exe", "moon_asm"})
	a.moonTGF = a.findTool([]string{"moon_extractTGF.exe", "moon_extractTGF"})
	a.moonScripts = a.findDir([]string{"moon_scripts"})
	a.oneHookDir = a.findDir([]string{"one_hook", "Hook_v5.1"})
}

func (a *App) findLCSETool() {
	a.lcsePath = a.findTool(executableNames())
}

func (a *App) findTool(names []string) string {
	checked := make(map[string]bool)

	addDir := func(d string, dirs *[]string) {
		if d == "" {
			return
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if checked[strings.ToLower(abs)] {
			return
		}
		checked[strings.ToLower(abs)] = true
		*dirs = append(*dirs, abs)
		bin := filepath.Join(abs, "bin")
		if !checked[strings.ToLower(bin)] {
			checked[strings.ToLower(bin)] = true
			*dirs = append(*dirs, bin)
		}
	}

	dirs := []string{}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for d, depth := exeDir, 0; depth < 4 && d != "." && d != string(filepath.Separator); depth++ {
			addDir(d, &dirs)
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for d, depth := cwd, 0; depth < 4 && d != "." && d != string(filepath.Separator); depth++ {
			addDir(d, &dirs)
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}

	for _, dir := range dirs {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if fileExists(candidate) {
				return candidate
			}
		}
	}

	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func (a *App) findDir(names []string) string {
	checked := make(map[string]bool)

	addDir := func(d string, dirs *[]string) {
		if d == "" {
			return
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if checked[strings.ToLower(abs)] {
			return
		}
		checked[strings.ToLower(abs)] = true
		*dirs = append(*dirs, abs)
		bin := filepath.Join(abs, "bin")
		if !checked[strings.ToLower(bin)] {
			checked[strings.ToLower(bin)] = true
			*dirs = append(*dirs, bin)
		}
	}

	dirs := []string{}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for d, depth := exeDir, 0; depth < 4 && d != "." && d != string(filepath.Separator); depth++ {
			addDir(d, &dirs)
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for d, depth := cwd, 0; depth < 4 && d != "." && d != string(filepath.Separator); depth++ {
			addDir(d, &dirs)
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}

	for _, dir := range dirs {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if dirExists(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func executableNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"lcse-tool.exe", "lcse-tool"}
	}
	return []string{"lcse-tool", "lcse-tool.exe"}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (a *App) GetLCSEPath() string {
	return a.lcsePath
}

func (a *App) GetToolPaths() map[string]string {
	return map[string]string{
		"lcse":        a.lcsePath,
		"moonAsm":     a.moonAsm,
		"moonTGF":     a.moonTGF,
		"moonScripts": a.moonScripts,
		"oneHook":     a.oneHookDir,
	}
}

func (a *App) SetLCSEPath() string {
	file, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Localiser lcse-tool",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Executable", Pattern: "*.exe;lcse-tool"},
			{DisplayName: "Tous les fichiers", Pattern: "*.*"},
		},
	})
	if err != nil || file == "" {
		return a.lcsePath
	}
	a.lcsePath = file
	return a.lcsePath
}

func (a *App) SelectArchiveBase() string {
	file, _ := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Selectionner lcsebody / archive de base",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Archive LCSE", Pattern: "lcsebody*;*.*"},
			{DisplayName: "Tous les fichiers", Pattern: "*.*"},
		},
	})
	if strings.HasSuffix(strings.ToLower(file), ".lst") {
		file = strings.TrimSuffix(file, filepath.Ext(file))
	}
	return file
}

func (a *App) SelectSNXFile() string {
	return a.selectFile("Selectionner un fichier SNX", "*.snx;*.SNX", "Fichiers SNX")
}

func (a *App) SelectTXTFile() string {
	return a.selectFile("Selectionner un fichier TXT", "*.txt", "Fichiers texte")
}

func (a *App) SelectTSVFile() string {
	return a.selectFile("Selectionner un fichier TSV", "*.tsv;*.txt", "Fichiers TSV/TXT")
}

func (a *App) SelectAnyFile(title string) string {
	return a.selectFile(title, "*.*", "Tous les fichiers")
}

func (a *App) SelectDirectory(title string) string {
	dir, _ := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{Title: title})
	return dir
}

func (a *App) SelectSaveFile(title string, defaultName string, pattern string, desc string) string {
	file, _ := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: desc, Pattern: pattern},
			{DisplayName: "Tous les fichiers", Pattern: "*.*"},
		},
	})
	return file
}

func (a *App) selectFile(title string, pattern string, desc string) string {
	file, _ := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: desc, Pattern: pattern},
			{DisplayName: "Tous les fichiers", Pattern: "*.*"},
		},
	})
	return file
}

func (a *App) log(msg string) {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "log", msg)
		return
	}
	// Keep command-line/integration runs diagnosable when no Wails context is
	// active. Regular GUI builds continue to emit only through the UI event.
	fmt.Println(msg)
}

func (a *App) logOK(msg string) {
	a.log("[OK] " + msg)
}

func (a *App) logError(msg string) {
	a.log("[ERROR] " + msg)
}

func (a *App) runLCSE(args ...string) error {
	if a.lcsePath == "" {
		a.logError("lcse-tool introuvable. Place lcse-tool.exe a cote de la GUI ou localise-le via la barre du haut.")
		return fmt.Errorf("lcse-tool not found")
	}
	return a.runExecutable(a.lcsePath, "", args...)
}

func (a *App) runExecutable(exe string, workdir string, args ...string) error {
	a.log("> " + filepath.Base(exe) + " " + strings.Join(args, " "))

	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.mu.Lock()
	a.cancelFunc = cancel
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		a.cancelFunc = nil
		a.mu.Unlock()
	}()

	cmd := exec.CommandContext(ctx, exe, args...)
	if workdir != "" {
		cmd.Dir = workdir
	}
	hideWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		a.logError(err.Error())
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		a.logError(err.Error())
		return err
	}

	if err := cmd.Start(); err != nil {
		a.logError(fmt.Sprintf("Demarrage impossible: %v", err))
		return err
	}

	done := make(chan struct{}, 2)
	stream := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			a.log(scanner.Text())
		}
		done <- struct{}{}
	}
	go stream(stdout)
	go stream(stderr)
	<-done
	<-done

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			a.log("[STOP] Operation interrompue.")
			return fmt.Errorf("cancelled")
		}
		a.logError(fmt.Sprintf("Echec: %v", err))
		return err
	}
	return nil
}

func (a *App) StopProcess() {
	a.mu.Lock()
	cancel := a.cancelFunc
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *App) UnpackArchive(archiveBase, outputDir, key, snxKey string) string {
	if archiveBase == "" || outputDir == "" {
		a.logError("Archive et dossier de sortie requis.")
		return "ERROR"
	}
	a.logSection("UNPACK ARCHIVE")
	args := withKeys([]string{"unpack", archiveBase, outputDir}, key, snxKey)
	if err := a.runLCSE(args...); err != nil {
		return "ERROR"
	}
	a.logOK("Archive extraite.")
	return "OK"
}

func (a *App) SNXToTXT(input, output string) string {
	if input == "" {
		a.logError("Fichier ou dossier SNX requis.")
		return "ERROR"
	}
	a.logSection("SNX -> TXT")
	args := []string{"snx2txt", input}
	if output != "" {
		args = append(args, output)
	}
	if err := a.runLCSE(args...); err != nil {
		return "ERROR"
	}
	a.logOK("Conversion terminee.")
	return "OK"
}

func (a *App) TXTToSNX(txtFile, originalSNX, outputSNX string) string {
	if txtFile == "" || originalSNX == "" {
		a.logError("TXT traduit et SNX original requis.")
		return "ERROR"
	}
	a.logSection("TXT -> SNX")
	args := []string{"txt2snx", txtFile, originalSNX}
	if outputSNX != "" {
		args = append(args, outputSNX)
	}
	if err := a.runLCSE(args...); err != nil {
		return "ERROR"
	}
	a.logOK("SNX reconstruit.")
	return "OK"
}

func (a *App) TXTToSNXBatch(txtDir, snxDir, outputDir string) string {
	if txtDir == "" || snxDir == "" {
		a.logError("Dossier TXT et dossier SNX originaux requis.")
		return "ERROR"
	}
	a.logSection("TXT -> SNX BATCH")
	args := []string{"txt2snx-batch", txtDir, snxDir}
	if outputDir != "" {
		args = append(args, outputDir)
	}
	if err := a.runLCSE(args...); err != nil {
		return "ERROR"
	}
	a.logOK("Batch SNX termine.")
	return "OK"
}

func (a *App) PatchArchive(originalBase, patchesDir, outputBase, key, snxKey string) string {
	if originalBase == "" || patchesDir == "" || outputBase == "" {
		a.logError("Archive originale, dossier patched et sortie requis.")
		return "ERROR"
	}
	a.logSection("PATCH ARCHIVE")
	args := withKeys([]string{"patch", originalBase, patchesDir, outputBase}, key, snxKey)
	if err := a.runLCSE(args...); err != nil {
		return "ERROR"
	}
	a.logOK("Archive patchee.")
	return "OK"
}

func (a *App) PackArchive(inputDir, outputBase, key, snxKey string) string {
	if inputDir == "" || outputBase == "" {
		a.logError("Dossier source et sortie requis.")
		return "ERROR"
	}
	a.logSection("PACK ARCHIVE")
	args := withKeys([]string{"pack", inputDir, outputBase}, key, snxKey)
	if err := a.runLCSE(args...); err != nil {
		return "ERROR"
	}
	a.logOK("Archive creee.")
	return "OK"
}

func (a *App) GetOneHookConfig() map[string]string {
	if a.oneHookDir == "" {
		a.oneHookDir = a.findDir([]string{"one_hook", "Hook_v5.1"})
	}
	fontName, debugLog := readHookINI(filepath.Join(a.oneHookDir, "lcse_hook.ini"))
	return map[string]string{
		"hookDir":  a.oneHookDir,
		"fontName": fontName,
		"debugLog": debugLog,
	}
}

func (a *App) SaveOneHookConfig(fontName, debugLog string) string {
	if a.oneHookDir == "" {
		a.oneHookDir = a.findDir([]string{"one_hook", "Hook_v5.1"})
	}
	if a.oneHookDir == "" {
		a.logError("Kit hook introuvable dans bin/one_hook.")
		return "ERROR"
	}
	if strings.TrimSpace(fontName) == "" {
		fontName = "MS Gothic"
	}
	if strings.TrimSpace(debugLog) == "" {
		debugLog = "0"
	}
	if err := writeHookINI(filepath.Join(a.oneHookDir, "lcse_hook.ini"), fontName, debugLog); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	a.logOK("Parametres du hook enregistres.")
	return "OK"
}

func (a *App) InstallOneHook(gameDir, fontName, debugLog string) string {
	if gameDir == "" {
		a.logError("Dossier du jeu ONE requis.")
		return "ERROR"
	}
	if a.SaveOneHookConfig(fontName, debugLog) != "OK" {
		return "ERROR"
	}
	a.logSection("INSTALL HOOK ONE")
	files := []string{"lcse_launcher.exe", "lcse_hook.dll", "lcse_hook.ini"}
	for _, name := range files {
		src := filepath.Join(a.oneHookDir, name)
		if !fileExists(src) {
			a.logError(fmt.Sprintf("%s introuvable dans le kit hook.", name))
			return "ERROR"
		}
		dst := filepath.Join(gameDir, name)
		if err := copyFile(src, dst); err != nil {
			a.logError(err.Error())
			return "ERROR"
		}
		a.log(fmt.Sprintf("%s -> %s", name, gameDir))
	}
	a.logOK("Kit hook ONE installe.")
	return "OK"
}

func (a *App) InstallMoonHook(gameDir, fontName, debugLog string) string {
	if gameDir == "" {
		a.logError("Dossier du jeu MOON requis.")
		return "ERROR"
	}
	if !fileExists(filepath.Join(gameDir, "MOON_eng.EXE")) {
		a.logError("MOON_eng.EXE introuvable dans le dossier choisi.")
		return "ERROR"
	}
	if a.SaveOneHookConfig(fontName, debugLog) != "OK" {
		return "ERROR"
	}
	a.logSection("INSTALL HOOK MOON")
	files := []struct {
		source string
		target string
	}{
		{source: "moon_launcher.exe", target: "MOON_fr.exe"},
		{source: "lcse_hook.dll", target: "lcse_hook.dll"},
		{source: "lcse_hook.ini", target: "lcse_hook.ini"},
	}
	for _, file := range files {
		src := filepath.Join(a.oneHookDir, file.source)
		if !fileExists(src) {
			a.logError(fmt.Sprintf("%s introuvable dans le kit hook.", file.source))
			return "ERROR"
		}
		if err := copyFile(src, filepath.Join(gameDir, file.target)); err != nil {
			a.logError(err.Error())
			return "ERROR"
		}
		a.log(fmt.Sprintf("%s -> %s", file.target, gameDir))
	}
	a.logOK("Kit hook MOON installe. Lance MOON_fr.exe ; MOON_eng.EXE reste le vrai executable du moteur.")
	return "OK"
}

func (a *App) MoonUnpackArchive(archiveBase, outputDir string) string {
	if archiveBase == "" || outputDir == "" {
		a.logError("Archive MOON et dossier de sortie requis.")
		return "ERROR"
	}
	a.logSection("MOON UNPACK")
	if err := a.runLCSE("unpack", archiveBase, outputDir); err != nil {
		return "ERROR"
	}
	a.logOK("Archive MOON extraite.")
	return "OK"
}

func (a *App) MoonConvertImages(inputPath, outputDir string) BatchResult {
	if inputPath == "" || outputDir == "" {
		a.logError("Source MOON et dossier PNG requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	info, err := os.Stat(inputPath)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(inputPath))
		if ext != ".tgf" && ext != ".bmp" {
			return a.MoonExtractImagesFromArchive(inputPath, outputDir)
		}
	}
	a.logSection("MOON IMAGES -> PNG")
	return a.convertMoonImageFiles(inputPath, outputDir)
}

func (a *App) MoonExtractImagesFromArchive(archiveBase, outputDir string) BatchResult {
	if archiveBase == "" || outputDir == "" {
		a.logError("Archive MOON et dossier PNG requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	if !fileExists(archiveBase) || !fileExists(archiveBase+".lst") {
		a.logError("Archive MOON ou fichier .lst introuvable.")
		return BatchResult{Status: "ERROR", Detail: "Archive invalide"}
	}

	a.logSection("MOON ARCHIVE -> PNG")
	tmp, err := os.MkdirTemp("", "lcse-moon-images-*")
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	defer os.RemoveAll(tmp)
	unpacked := filepath.Join(tmp, "unpacked")
	if err := a.runLCSE("unpack", archiveBase, unpacked); err != nil {
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	return a.convertMoonImageFiles(unpacked, outputDir)
}

func (a *App) MoonScriptsToUTF(inputDir, outputDir string) BatchResult {
	if inputDir == "" || outputDir == "" {
		a.logError("Dossier scripts source et sortie UTF-8 requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("MOON SCRIPTS -> UTF-8")
	files, err := txtFiles(inputDir)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	if len(files) == 0 {
		msg := "Aucun script .txt trouve dans le dossier source."
		if countFilesWithExt(inputDir, ".snx") > 0 {
			msg = "Ce dossier contient des SNX MOON. Utilise MOON SNX vers TXT pour les desassembler directement en UTF-8."
		}
		a.logError(msg)
		return BatchResult{Status: "ERROR", Detail: msg}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	done := 0
	for _, file := range files {
		text, err := readTextAuto(file)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(file), err))
			continue
		}
		out := filepath.Join(outputDir, filepath.Base(file))
		if err := writeUTF8BOM(out, normalizeNewlines(text)); err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(file), err))
			continue
		}
		done++
	}
	msg := fmt.Sprintf("%d scripts UTF-8 crees", done)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) MoonBundledScriptsToUTF(outputDir string) BatchResult {
	if outputDir == "" {
		a.logError("Dossier de sortie UTF-8 requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.moonScripts = a.resolveMoonScripts("")
	if a.moonScripts == "" {
		msg := moonScriptsMissingMessage()
		a.logError(msg)
		return BatchResult{Status: "ERROR", Detail: msg}
	}
	a.log(fmt.Sprintf("Sources MOON: %s", a.moonScripts))
	return a.MoonScriptsToUTF(a.moonScripts, outputDir)
}

func (a *App) MoonSNXToTXT(inputPath, outputPath string) string {
	if inputPath == "" {
		a.logError("Fichier ou dossier SNX MOON requis.")
		return "ERROR"
	}
	info, err := os.Stat(inputPath)
	if err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	a.logSection("MOON SNX -> TXT")
	a.log("Desassemblage direct du bytecode MOON vers des scripts UTF-8.")
	if info.IsDir() {
		if outputPath == "" {
			outputPath = inputPath + "_txt"
		}
	}
	if outputPath == "" {
		outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".txt"
	}
	if err := a.runLCSE("snx2txt", inputPath, outputPath); err != nil {
		return "ERROR"
	}
	a.logOK("Scripts MOON UTF-8 crees.")
	return "OK"
}

func moonScriptsMissingMessage() string {
	return "Sources optionnelles du MOON Kit introuvables. Pour un SNX extrait d'une archive, utilise la conversion directe MOON SNX vers TXT."
}

func isMoonScriptsDir(path string) bool {
	if !dirExists(path) {
		return false
	}
	for _, name := range []string{"INIT.txt", "A_D2.txt", "DAY01A.txt"} {
		if fileExists(filepath.Join(path, name)) {
			return true
		}
	}
	return false
}

func (a *App) resolveMoonScripts(inputPath string) string {
	if isMoonScriptsDir(a.moonScripts) {
		return a.moonScripts
	}
	if found := a.findDir([]string{"moon_scripts"}); isMoonScriptsDir(found) {
		return found
	}

	anchor := inputPath
	if anchor != "" {
		if info, err := os.Stat(anchor); err == nil && !info.IsDir() {
			anchor = filepath.Dir(anchor)
		}
	}

	candidates := []string{}
	addCandidates := func(dir string) {
		if dir == "" || dir == "." {
			return
		}
		candidates = append(candidates,
			dir,
			filepath.Join(dir, "moon_scripts"),
			filepath.Join(dir, "scripts"),
		)
	}
	addCandidates(anchor)
	if anchor != "" {
		parent := filepath.Dir(anchor)
		addCandidates(parent)
		if entries, err := os.ReadDir(parent); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					addCandidates(filepath.Join(parent, entry.Name()))
				}
			}
		}
	}

	seen := map[string]bool{}
	for _, candidate := range candidates {
		key := strings.ToLower(filepath.Clean(candidate))
		if seen[key] {
			continue
		}
		seen[key] = true
		if isMoonScriptsDir(candidate) {
			return candidate
		}
	}
	return ""
}

func (a *App) copyMoonScriptForSNX(snxPath, outputPath string) error {
	base := strings.TrimSuffix(filepath.Base(snxPath), filepath.Ext(snxPath))
	source := filepath.Join(a.moonScripts, base+".txt")
	if !fileExists(source) {
		return fmt.Errorf("source kit %s.txt introuvable", base)
	}
	text, err := readTextAuto(source)
	if err != nil {
		return err
	}
	if err := writeUTF8BOM(outputPath, normalizeNewlines(text)); err != nil {
		return err
	}
	a.log(fmt.Sprintf("%s -> %s", filepath.Base(snxPath), filepath.Base(outputPath)))
	return nil
}

func (a *App) MoonTXTToSNX(txtFile, outputSNX string) string {
	if txtFile == "" {
		a.logError("Fichier TXT MOON requis.")
		return "ERROR"
	}
	if a.moonAsm == "" {
		a.moonAsm = a.findTool([]string{"moon_asm.exe", "moon_asm"})
	}
	if a.moonAsm == "" {
		a.logError("moon_asm.exe introuvable dans bin/.")
		return "ERROR"
	}
	a.logSection("MOON TXT -> SNX")
	tmp, err := os.MkdirTemp("", "lcse-moon-asm-one-*")
	if err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	defer os.RemoveAll(tmp)
	tmpScripts := filepath.Join(tmp, "scripts")
	tmpPatch := filepath.Join(tmp, "patch")
	tmpFinal := filepath.Join(tmp, "final")
	if err := os.MkdirAll(tmpScripts, 0755); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	if err := os.MkdirAll(tmpPatch, 0755); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	text, err := readTextAuto(txtFile)
	if err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	tmpScript := filepath.Join(tmpScripts, filepath.Base(txtFile))
	if err := writeShiftJIS(tmpScript, normalizeNewlines(text)); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	if err := a.runExecutable(a.moonAsm, tmp); err != nil {
		return "ERROR"
	}
	base := strings.TrimSuffix(filepath.Base(txtFile), filepath.Ext(txtFile))
	built := filepath.Join(tmpPatch, base+".SNX")
	if !fileExists(built) {
		built = filepath.Join(tmpPatch, base+".snx")
	}
	if !fileExists(built) {
		a.logError("SNX assemble introuvable dans le dossier patch temporaire.")
		return "ERROR"
	}
	finalized := filepath.Join(tmpFinal, filepath.Base(built))
	if err := a.runLCSE("moon-accents", built, finalized); err != nil {
		return "ERROR"
	}
	if outputSNX == "" {
		outputSNX = strings.TrimSuffix(txtFile, filepath.Ext(txtFile)) + ".snx"
	}
	if err := copyFile(finalized, outputSNX); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	a.logOK(fmt.Sprintf("%s cree", outputSNX))
	return "OK"
}

func (a *App) MoonExportDialogues(scriptsDir, outputDir string) BatchResult {
	if scriptsDir == "" || outputDir == "" {
		a.logError("Dossier scripts et dossier dialogue requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("MOON EXPORT DIALOGUES")
	files, err := moonActiveTxtFiles(scriptsDir)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	total, done := 0, 0
	for _, file := range files {
		count, err := exportMoonDialogueFile(file, outputDir)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(file), err))
			continue
		}
		if count > 0 {
			total += count
			done++
		}
	}
	msg := fmt.Sprintf("%d fichiers, %d lignes exportees", done, total)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) MoonImportDialogues(scriptsDir, dialoguesDir, outputDir string) BatchResult {
	if scriptsDir == "" || dialoguesDir == "" || outputDir == "" {
		a.logError("Scripts source, dialogues traduits et dossier de sortie requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("MOON IMPORT DIALOGUES")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	activeScripts, err := moonActiveTxtFiles(scriptsDir)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	for _, source := range activeScripts {
		if err := copyFile(source, filepath.Join(outputDir, filepath.Base(source))); err != nil {
			a.logError(err.Error())
			return BatchResult{Status: "ERROR", Detail: err.Error()}
		}
	}
	dialogueFiles, err := filepath.Glob(filepath.Join(dialoguesDir, "*.dlg.txt"))
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	sort.Strings(dialogueFiles)
	total, done, skipped := 0, 0, 0
	for _, dialogueFile := range dialogueFiles {
		base := strings.TrimSuffix(filepath.Base(dialogueFile), ".dlg.txt")
		source := filepath.Join(scriptsDir, base+".txt")
		if !fileExists(source) {
			a.log(fmt.Sprintf("[SKIP] %s: script source introuvable", filepath.Base(dialogueFile)))
			skipped++
			continue
		}
		out := filepath.Join(outputDir, base+".txt")
		count, err := importMoonDialogueFile(source, dialogueFile, out)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(dialogueFile), err))
			skipped++
			continue
		}
		total += count
		done++
	}
	msg := fmt.Sprintf("%d scripts complets, %d fichiers dialogues, %d lignes importees, %d ignores", len(activeScripts), done, total, skipped)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) OneExportDialogues(scriptsDir, outputDir string) BatchResult {
	if scriptsDir == "" || outputDir == "" {
		a.logError("Dossier scripts ONE et dossier dialogue requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("ONE EXPORT DIALOGUES")
	files, err := txtFiles(scriptsDir)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	total, done := 0, 0
	for _, file := range files {
		count, err := exportOneDialogueFile(file, outputDir)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(file), err))
			continue
		}
		if count > 0 {
			total += count
			done++
		}
	}
	msg := fmt.Sprintf("%d fichiers, %d lignes exportees", done, total)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) OneImportDialogues(scriptsDir, dialoguesDir, outputDir string) BatchResult {
	if scriptsDir == "" || dialoguesDir == "" || outputDir == "" {
		a.logError("Scripts source, dialogues traduits et dossier de sortie requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("ONE IMPORT DIALOGUES")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	dialogueFiles, err := filepath.Glob(filepath.Join(dialoguesDir, "*.dlg.txt"))
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	sort.Strings(dialogueFiles)
	total, done, skipped := 0, 0, 0
	for _, dialogueFile := range dialogueFiles {
		base := strings.TrimSuffix(filepath.Base(dialogueFile), ".dlg.txt")
		source := filepath.Join(scriptsDir, base+".txt")
		if !fileExists(source) {
			a.log(fmt.Sprintf("[SKIP] %s: script source introuvable", filepath.Base(dialogueFile)))
			skipped++
			continue
		}
		out := filepath.Join(outputDir, base+".txt")
		count, err := importOneDialogueFile(source, dialogueFile, out)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(dialogueFile), err))
			skipped++
			continue
		}
		total += count
		done++
	}
	msg := fmt.Sprintf("%d fichiers, %d lignes importees, %d ignores", done, total, skipped)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) MoonAssembleScripts(scriptsDir, outputPatchDir string) string {
	if scriptsDir == "" || outputPatchDir == "" {
		a.logError("Dossier scripts et dossier patch SNX requis.")
		return "ERROR"
	}
	if a.moonAsm == "" {
		a.moonAsm = a.findTool([]string{"moon_asm.exe", "moon_asm"})
	}
	if a.moonAsm == "" {
		a.logError("moon_asm.exe introuvable dans bin/.")
		return "ERROR"
	}
	a.logSection("MOON ASSEMBLE SCRIPTS")
	tmp, err := os.MkdirTemp("", "lcse-moon-asm-*")
	if err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	defer os.RemoveAll(tmp)
	tmpScripts := filepath.Join(tmp, "scripts")
	tmpPatch := filepath.Join(tmp, "patch")
	tmpFinal := filepath.Join(tmp, "final")
	if err := os.MkdirAll(tmpScripts, 0755); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	if err := os.MkdirAll(tmpPatch, 0755); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	if err := copyScriptsAsShiftJIS(scriptsDir, tmpScripts); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	if err := a.runExecutable(a.moonAsm, tmp); err != nil {
		return "ERROR"
	}
	if err := a.runLCSE("moon-accents", tmpPatch, tmpFinal); err != nil {
		return "ERROR"
	}
	if err := os.MkdirAll(outputPatchDir, 0755); err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	count, err := copyFilesByExt(tmpFinal, outputPatchDir, ".snx")
	if err != nil {
		a.logError(err.Error())
		return "ERROR"
	}
	a.logOK(fmt.Sprintf("%d SNX assembles -> %s", count, outputPatchDir))
	return "OK"
}

func withKeys(args []string, key, snxKey string) []string {
	out := append([]string{}, args...)
	if strings.TrimSpace(key) != "" {
		out = append(out, "--key", strings.TrimSpace(key))
	}
	if strings.TrimSpace(snxKey) != "" {
		out = append(out, "--snxkey", strings.TrimSpace(snxKey))
	}
	return out
}

func (a *App) ExtractTranslations(inputTxt, outputTSV string) BatchResult {
	if inputTxt == "" || outputTSV == "" {
		a.logError("TXT source et TSV de sortie requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("EXTRACT TSV")
	count, err := extractTranslationFile(inputTxt, outputTSV)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	msg := fmt.Sprintf("%d lignes TXT extraites -> %s", count, outputTSV)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) ExtractTranslationsBatch(inputDir, outputDir string) BatchResult {
	if inputDir == "" || outputDir == "" {
		a.logError("Dossier TXT source et dossier TSV requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("EXTRACT TSV BATCH")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}

	files, err := txtFiles(inputDir)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	total, done, skipped := 0, 0, 0
	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		out := filepath.Join(outputDir, base+"_extracted.tsv")
		count, err := extractTranslationFile(file, out)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(file), err))
			skipped++
			continue
		}
		a.log(fmt.Sprintf("[%d] %s -> %s (%d)", done+1, filepath.Base(file), filepath.Base(out), count))
		total += count
		done++
	}
	msg := fmt.Sprintf("%d fichiers, %d lignes, %d ignores", done, total, skipped)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) ReinjectTranslations(originalTxt, translationTSV, outputTxt string) BatchResult {
	if originalTxt == "" || translationTSV == "" || outputTxt == "" {
		a.logError("TXT original, TSV traduit et TXT de sortie requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("REINJECT TSV")
	count, err := reinjectTranslationFile(originalTxt, translationTSV, outputTxt)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	msg := fmt.Sprintf("%d lignes reinjectees -> %s", count, outputTxt)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) ReinjectTranslationsBatch(originalDir, tsvDir, outputDir string) BatchResult {
	if originalDir == "" || tsvDir == "" || outputDir == "" {
		a.logError("Dossiers original, TSV et sortie requis.")
		return BatchResult{Status: "ERROR", Detail: "Parametres manquants"}
	}
	a.logSection("REINJECT TSV BATCH")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}

	tsvMatches, err := filepath.Glob(filepath.Join(tsvDir, "*_extracted.tsv"))
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	sort.Strings(tsvMatches)
	total, done, skipped := 0, 0, 0
	for _, tsv := range tsvMatches {
		name := strings.TrimSuffix(filepath.Base(tsv), "_extracted.tsv") + ".txt"
		original := filepath.Join(originalDir, name)
		if !fileExists(original) {
			a.log(fmt.Sprintf("[SKIP] %s: TXT original introuvable", filepath.Base(tsv)))
			skipped++
			continue
		}
		out := filepath.Join(outputDir, strings.TrimSuffix(name, filepath.Ext(name))+"_patched.txt")
		count, err := reinjectTranslationFile(original, tsv, out)
		if err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(tsv), err))
			skipped++
			continue
		}
		a.log(fmt.Sprintf("[%d] %s + %s -> %s (%d)", done+1, name, filepath.Base(tsv), filepath.Base(out), count))
		total += count
		done++
	}
	msg := fmt.Sprintf("%d fichiers, %d lignes, %d ignores", done, total, skipped)
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

func (a *App) logSection(title string) {
	a.log("========================================")
	a.log("  " + title)
	a.log("========================================")
}

func txtFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".txt") {
			continue
		}
		if strings.HasSuffix(lower, "_patched.txt") || strings.HasSuffix(lower, "_extracted.txt") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files, nil
}

func snxFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".snx") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func countFilesWithExt(dir, ext string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	ext = strings.ToLower(ext)
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(entry.Name())) == ext {
			count++
		}
	}
	return count
}

func extractTranslationFile(inputTxt, outputTSV string) (int, error) {
	content, err := readTextAuto(inputTxt)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(content, "\n")
	var out strings.Builder
	out.WriteString("index\toriginal\r\n")
	count := 0
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) >= 4 && parts[2] == "TXT" {
			out.WriteString(parts[0])
			out.WriteString("\t")
			out.WriteString(strings.ReplaceAll(parts[3], "\r", ""))
			out.WriteString("\r\n")
			count++
		}
	}
	return count, writeUTF8BOM(outputTSV, out.String())
}

func reinjectTranslationFile(originalTxt, translationTSV, outputTxt string) (int, error) {
	originalContent, err := readTextAuto(originalTxt)
	if err != nil {
		return 0, err
	}
	tsvContent, err := readTextAuto(translationTSV)
	if err != nil {
		return 0, err
	}

	translations := map[string]string{}
	tsvLines := strings.Split(tsvContent, "\n")
	for i, line := range tsvLines {
		line = strings.TrimRight(line, "\r")
		if i == 0 || line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			translations[parts[0]] = parts[1]
		}
	}

	originals := map[string]string{}
	srcLines := strings.Split(originalContent, "\n")
	for _, line := range srcLines {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) >= 4 && parts[2] == "TXT" {
			originals[parts[0]] = parts[3]
		}
	}

	for idx, text := range translations {
		if original, ok := originals[idx]; ok && strings.HasPrefix(text, "\"") && !strings.HasPrefix(original, "\"") {
			if !(strings.HasSuffix(text, "\"") && len(text) > 2) {
				translations[idx] = strings.TrimPrefix(text, "\"")
			}
		}
	}

	patchedCount := 0
	outLines := make([]string, 0, len(srcLines))
	for _, line := range srcLines {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			outLines = append(outLines, line)
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) >= 4 && parts[2] == "TXT" {
			if text, ok := translations[parts[0]]; ok {
				parts[3] = text
				outLines = append(outLines, strings.Join(parts, "\t"))
				patchedCount++
				continue
			}
		}
		outLines = append(outLines, line)
	}

	return patchedCount, writeUTF8BOM(outputTxt, strings.Join(outLines, "\r\n"))
}

func readTextAuto(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:]), nil
	}
	if utf8.Valid(data) && hasUTF8Signal(data) {
		return string(data), nil
	}
	reader := transform.NewReader(bytes.NewReader(data), japanese.ShiftJIS.NewDecoder())
	out, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func hasUTF8Signal(data []byte) bool {
	for _, b := range data {
		if b >= 0x80 {
			return true
		}
	}
	return false
}

func writeUTF8BOM(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(content)...)
	return os.WriteFile(path, data, 0644)
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func convertMoonBMPToPNG(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	img, err := decodeMoonBMP(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil && filepath.Dir(outputPath) != "." {
		return err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, img)
}

type moonDialogueLine struct {
	ID   int
	Line int
	Tag  string
	Text string
}

func parseMoonDialogueLine(line string) (string, string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "TEXT ") {
		return "TEXT", strings.TrimPrefix(trimmed, "TEXT "), true
	}
	if strings.HasPrefix(trimmed, "SETSTATUS ") {
		rest := strings.TrimPrefix(trimmed, "SETSTATUS ")
		start := strings.Index(rest, "'")
		end := strings.LastIndex(rest, "'")
		if start >= 0 && end > start {
			return "SETSTATUS", rest[start+1 : end], true
		}
	}
	return "", "", false
}

func exportMoonDialogueFile(scriptFile, outputDir string) (int, error) {
	text, err := readTextAuto(scriptFile)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var out strings.Builder
	out.WriteString("id\tline\ttag\ttext\r\n")
	count := 0
	for i, line := range lines {
		tag, value, ok := parseMoonDialogueLine(strings.TrimRight(line, "\r"))
		if !ok {
			continue
		}
		count++
		out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\r\n", count, i+1, tag, escapeMoonCell(value)))
	}
	if count == 0 {
		return 0, nil
	}
	base := strings.TrimSuffix(filepath.Base(scriptFile), filepath.Ext(scriptFile))
	return count, writeUTF8BOM(filepath.Join(outputDir, base+".dlg.txt"), out.String())
}

func importMoonDialogueFile(scriptFile, dialogueFile, outputFile string) (int, error) {
	scriptText, err := readTextAuto(scriptFile)
	if err != nil {
		return 0, err
	}
	dialogues, err := readMoonDialogueFile(dialogueFile)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.ReplaceAll(scriptText, "\r\n", "\n"), "\n")
	currentID := 0
	changed := 0
	for i, line := range lines {
		tag, _, ok := parseMoonDialogueLine(strings.TrimRight(line, "\r"))
		if !ok {
			continue
		}
		currentID++
		replacement, ok := dialogues[currentID]
		if !ok || replacement.Tag != tag {
			continue
		}
		prefixLen := len(line) - len(strings.TrimLeft(line, " \t"))
		prefix := line[:prefixLen]
		if tag == "TEXT" {
			lines[i] = prefix + "TEXT " + replacement.Text
		} else {
			trimmed := strings.TrimLeft(line, " \t")
			rest := strings.TrimPrefix(trimmed, "SETSTATUS ")
			start := strings.Index(rest, "'")
			end := strings.LastIndex(rest, "'")
			if start < 0 || end <= start {
				continue
			}
			beforeValue := prefix + "SETSTATUS " + rest[:start+1]
			afterValue := rest[end:]
			lines[i] = beforeValue + escapeMoonQuotedValue(replacement.Text) + afterValue
		}
		changed++
	}
	return changed, writeUTF8BOM(outputFile, normalizeNewlines(strings.Join(lines, "\n")))
}

func escapeMoonQuotedValue(value string) string {
	var out strings.Builder
	out.Grow(len(value))
	for index := 0; index < len(value); index++ {
		if value[index] == '\\' && index+1 < len(value) && value[index+1] == '\'' {
			out.WriteString("\\'")
			index++
			continue
		}
		if value[index] == '\'' {
			out.WriteString("\\'")
			continue
		}
		out.WriteByte(value[index])
	}
	return out.String()
}

func exportOneDialogueFile(scriptFile, outputDir string) (int, error) {
	text, err := readTextAuto(scriptFile)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var out strings.Builder
	out.WriteString("id\tline\ttag\ttext\r\n")
	count := 0
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 || parts[2] != "TXT" {
			continue
		}
		count++
		out.WriteString(fmt.Sprintf("%d\t%d\tTXT\t%s\r\n", count, i+1, escapeMoonCell(parts[3])))
	}
	if count == 0 {
		return 0, nil
	}
	base := strings.TrimSuffix(filepath.Base(scriptFile), filepath.Ext(scriptFile))
	return count, writeUTF8BOM(filepath.Join(outputDir, base+".dlg.txt"), out.String())
}

func importOneDialogueFile(scriptFile, dialogueFile, outputFile string) (int, error) {
	scriptText, err := readTextAuto(scriptFile)
	if err != nil {
		return 0, err
	}
	dialogues, err := readMoonDialogueFile(dialogueFile)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.ReplaceAll(scriptText, "\r\n", "\n"), "\n")
	currentID := 0
	changed := 0
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 || parts[2] != "TXT" {
			continue
		}
		currentID++
		replacement, ok := dialogues[currentID]
		if !ok || replacement.Tag != "TXT" {
			continue
		}
		parts[3] = replacement.Text
		lines[i] = strings.Join(parts, "\t")
		changed++
	}
	return changed, writeUTF8BOM(outputFile, normalizeNewlines(strings.Join(lines, "\n")))
}

func readMoonDialogueFile(path string) (map[int]moonDialogueLine, error) {
	text, err := readTextAuto(path)
	if err != nil {
		return nil, err
	}
	result := map[int]moonDialogueLine{}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		var id, lineNo int
		fmt.Sscanf(parts[0], "%d", &id)
		fmt.Sscanf(parts[1], "%d", &lineNo)
		if id <= 0 {
			continue
		}
		result[id] = moonDialogueLine{
			ID:   id,
			Line: lineNo,
			Tag:  parts[2],
			Text: unescapeMoonCell(parts[3]),
		}
	}
	return result, nil
}

func escapeMoonCell(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func unescapeMoonCell(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			out.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			out.WriteByte('\n')
		case 't':
			out.WriteByte('\t')
		case '\\':
			out.WriteByte('\\')
		default:
			out.WriteByte('\\')
			out.WriteByte(s[i])
		}
	}
	return out.String()
}

func copyScriptsAsShiftJIS(inputDir, outputDir string) error {
	files, err := moonActiveTxtFiles(inputDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		text, err := readTextAuto(file)
		if err != nil {
			return err
		}
		out := filepath.Join(outputDir, filepath.Base(file))
		if err := writeShiftJIS(out, normalizeNewlines(text)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(file), err)
		}
	}
	return nil
}

var moonLegacySplitScripts = map[string][]string{
	"DAY01":  {"DAY01A", "DAY01B"},
	"DAY02":  {"DAY02A", "DAY02B"},
	"DAY03":  {"DAY03A", "DAY03B"},
	"DAY07T": {"DAY07TA", "DAY07TB"},
	"DAY08":  {"DAY08A", "DAY08B"},
	"DAY20":  {"DAY20A", "DAY20B"},
}

// English MOON archives retain six encrypted, unsplit Japanese scripts next
// to the active A/B replacements. The originals exceed the 64 KiB VM limit
// after an English/French rebuild, so exclude them whenever both replacements
// are present. They remain available in the raw disassembly for auditing.
func moonActiveTxtFiles(dir string) ([]string, error) {
	files, err := txtFiles(dir)
	if err != nil {
		return nil, err
	}
	present := map[string]bool{}
	for _, file := range files {
		present[strings.ToUpper(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))] = true
	}
	active := make([]string, 0, len(files))
	for _, file := range files {
		base := strings.ToUpper(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))
		parts, legacy := moonLegacySplitScripts[base]
		if legacy {
			allParts := true
			for _, part := range parts {
				if !present[part] {
					allParts = false
					break
				}
			}
			if allParts {
				continue
			}
		}
		active = append(active, file)
	}
	return active, nil
}

var moonAccentToSentinel = map[rune]byte{
	'é': 0x02, 'è': 0x03, 'ç': 0x04,
	'à': 0x06, 'â': 0x07, 'û': 0x08,
	'ô': 0x0E, 'ê': 0x0F, 'î': 0x10,
	'ù': 0x11, 'ë': 0x12, 'ï': 0x13,
	'ü': 0x14,
}

var moonAccentFallback = map[rune]byte{
	'À': 'A', 'Â': 'A', 'Ç': 'C', 'È': 'E', 'É': 'E', 'Ê': 'E',
	'Î': 'I', 'Ô': 'O', 'Ù': 'U', 'Û': 'U', 'Œ': 'O', 'œ': 'o',
}

func encodeMoonAssemblerSource(content string) ([]byte, error) {
	var out bytes.Buffer
	values := []rune(content)
	atLineStart := true
	escapeShiftJISTrail := false
	for index := 0; index < len(values); index++ {
		value := values[index]
		if atLineStart {
			lineEnd := index
			for lineEnd < len(values) && values[lineEnd] != '\n' {
				lineEnd++
			}
			line := strings.TrimLeft(string(values[index:lineEnd]), "\ufeff \t\r")
			escapeShiftJISTrail = strings.HasPrefix(line, "TEXT ")
			atLineStart = false
		}
		if value == '\\' && index+1 < len(values) {
			if values[index+1] == '\\' {
				out.WriteString("\\\\")
				index++
				continue
			}
			if values[index+1] == 'n' {
				// 0x01 breaks read from an existing SNX are soft layout breaks.
				// Let moon_asm recompute wrapping for the translated text instead
				// of forcing the old English/Japanese layout.
				if index > 0 && index+2 < len(values) && values[index-1] <= 0xFF && values[index+2] <= 0xFF {
					// In Latin text, the soft break also separates two words.
					out.WriteByte(' ')
				}
				index++
				continue
			}
		}
		if sentinel, ok := moonAccentToSentinel[value]; ok {
			out.WriteByte(sentinel)
			continue
		}
		if fallback, ok := moonAccentFallback[value]; ok {
			out.WriteByte(fallback)
			continue
		}
		reader := transform.NewReader(strings.NewReader(string(value)), japanese.ShiftJIS.NewEncoder())
		encoded, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("caractere non pris en charge U+%04X: %w", value, err)
		}
		// moon_asm's TEXT parser treats 0x5C as an escape byte even when it is
		// the trail byte of a Shift-JIS character (for example 構 = 8D 5C).
		// Doubling that trail byte makes the assembler emit the original pair;
		// quoted command arguments do not need this workaround.
		if escapeShiftJISTrail && len(encoded) == 2 && encoded[1] == 0x5C {
			out.WriteByte(encoded[0])
			out.WriteByte(encoded[1])
			out.WriteByte(encoded[1])
			continue
		}
		out.Write(encoded)
		if value == '\n' {
			atLineStart = true
		}
	}
	return out.Bytes(), nil
}

func writeShiftJIS(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	data, err := encodeMoonAssemblerSource(content)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func copyFilesByExt(inputDir, outputDir, ext string) (int, error) {
	count := 0
	err := filepath.WalkDir(inputDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != strings.ToLower(ext) {
			return nil
		}
		rel, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}
		out := filepath.Join(outputDir, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil && filepath.Dir(dst) != "." {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func readHookINI(path string) (string, string) {
	fontName := "MS Gothic"
	debugLog := "0"
	data, err := os.ReadFile(path)
	if err != nil {
		return fontName, debugLog
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			fontName = strings.TrimSpace(strings.TrimPrefix(line, "Name="))
		}
		if strings.HasPrefix(line, "Log=") {
			debugLog = strings.TrimSpace(strings.TrimPrefix(line, "Log="))
		}
	}
	return fontName, debugLog
}

func writeHookINI(path, fontName, debugLog string) error {
	content := "[Font]\r\nName=" + strings.TrimSpace(fontName) + "\r\n\r\n[Debug]\r\nLog=" + strings.TrimSpace(debugLog) + "\r\n"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
