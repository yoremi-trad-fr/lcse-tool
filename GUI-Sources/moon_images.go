package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type moonImageFile struct {
	path string
	rel  string
}

func (a *App) convertMoonImageFiles(inputPath, outputDir string) BatchResult {
	info, err := os.Stat(inputPath)
	if err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}

	baseRoot := filepath.Dir(inputPath)
	files := []string{}
	if info.IsDir() {
		baseRoot = inputPath
		err = filepath.WalkDir(inputPath, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".tgf" || ext == ".bmp" {
					files = append(files, path)
				}
			}
			return nil
		})
		if err != nil {
			a.logError(err.Error())
			return BatchResult{Status: "ERROR", Detail: err.Error()}
		}
	} else {
		ext := strings.ToLower(filepath.Ext(inputPath))
		if ext != ".tgf" && ext != ".bmp" {
			msg := "Fichier TGF ou BMP requis."
			a.logError(msg)
			return BatchResult{Status: "ERROR", Detail: msg}
		}
		files = append(files, inputPath)
	}
	if len(files) == 0 {
		msg := "Aucun fichier TGF ou BMP dans la source."
		a.logError(msg)
		return BatchResult{Status: "ERROR", Detail: msg}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}

	sort.Strings(files)
	tgfFiles, bmpFiles := []moonImageFile{}, []moonImageFile{}
	for _, path := range files {
		rel, err := filepath.Rel(baseRoot, path)
		if err != nil {
			a.logError(err.Error())
			return BatchResult{Status: "ERROR", Detail: err.Error()}
		}
		image := moonImageFile{path: path, rel: strings.TrimSuffix(rel, filepath.Ext(rel)) + ".png"}
		if strings.EqualFold(filepath.Ext(path), ".tgf") {
			tgfFiles = append(tgfFiles, image)
		} else {
			bmpFiles = append(bmpFiles, image)
		}
	}

	// A BMP of the same name is the translated replacement for a TGF.
	bmpOutputs := make(map[string]bool, len(bmpFiles))
	for _, file := range bmpFiles {
		bmpOutputs[strings.ToLower(file.rel)] = true
	}
	filtered := tgfFiles[:0]
	for _, file := range tgfFiles {
		if !bmpOutputs[strings.ToLower(file.rel)] {
			filtered = append(filtered, file)
		}
	}
	tgfFiles = filtered

	converted, skipped := 0, 0
	convert := func(source string, file moonImageFile) {
		if err := convertMoonBMPToPNG(source, filepath.Join(outputDir, file.rel)); err != nil {
			a.log(fmt.Sprintf("[SKIP] %s: %v", filepath.Base(file.path), err))
			skipped++
			return
		}
		converted++
		if converted <= 25 || converted%100 == 0 {
			a.log(fmt.Sprintf("[%d] %s -> %s", converted, filepath.Base(file.path), file.rel))
		}
	}

	if len(tgfFiles) > 0 {
		if a.moonTGF == "" {
			a.moonTGF = a.findTool([]string{"moon_extractTGF.exe", "moon_extractTGF"})
		}
		if a.moonTGF == "" {
			msg := "moon_extractTGF.exe introuvable dans bin/."
			a.logError(msg)
			return BatchResult{Status: "ERROR", Detail: msg}
		}
		tmp, err := os.MkdirTemp("", "lcse-moon-tgf-*")
		if err != nil {
			a.logError(err.Error())
			return BatchResult{Status: "ERROR", Detail: err.Error()}
		}
		defer os.RemoveAll(tmp)
		if err := writeMoonTGFArchive(tmp, tgfFiles); err != nil {
			a.logError(err.Error())
			return BatchResult{Status: "ERROR", Detail: err.Error()}
		}
		if err := a.runExecutable(a.moonTGF, tmp); err != nil {
			return BatchResult{Status: "ERROR", Detail: err.Error()}
		}
		for i, file := range tgfFiles {
			decoded := filepath.Join(tmp, "TGF", fmt.Sprintf("IMG%06d.BMP", i+1))
			convert(decoded, file)
		}
	}
	for _, file := range bmpFiles {
		convert(file.path, file)
	}

	msg := fmt.Sprintf("%d PNG crees, %d images ignorees", converted, skipped)
	if skipped > 0 {
		a.logError(msg)
		return BatchResult{Status: "ERROR", Detail: msg}
	}
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}

// The original MOON extractor decodes TGF entries from a moon/moon.lst pair.
// Give each input a unique temporary name so files from different folders
// cannot overwrite one another in its flat TGF output directory.
func writeMoonTGFArchive(dir string, files []moonImageFile) error {
	if len(files) > 999999 {
		return fmt.Errorf("trop de fichiers TGF: %d", len(files))
	}
	if err := os.MkdirAll(filepath.Join(dir, "TGF"), 0755); err != nil {
		return err
	}
	archive, err := os.Create(filepath.Join(dir, "moon"))
	if err != nil {
		return err
	}
	defer archive.Close()
	list, err := os.Create(filepath.Join(dir, "moon.lst"))
	if err != nil {
		return err
	}
	defer list.Close()

	const xorKey uint32 = 0xCCCCCCCC
	if err := binary.Write(list, binary.LittleEndian, uint32(len(files))^xorKey); err != nil {
		return err
	}
	var offset uint64
	for i, file := range files {
		input, err := os.Open(file.path)
		if err != nil {
			return err
		}
		info, err := input.Stat()
		if err != nil {
			input.Close()
			return err
		}
		if info.Size() < 0 || uint64(info.Size()) > math.MaxUint32 || offset+uint64(info.Size()) > math.MaxUint32 {
			input.Close()
			return fmt.Errorf("archive TGF temporaire trop grande")
		}
		count, copyErr := io.Copy(archive, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if count != info.Size() {
			return fmt.Errorf("lecture incomplete de %s", file.path)
		}
		if err := binary.Write(list, binary.LittleEndian, uint32(offset)^xorKey); err != nil {
			return err
		}
		if err := binary.Write(list, binary.LittleEndian, uint32(count)^xorKey); err != nil {
			return err
		}
		name := fmt.Sprintf("IMG%06d.TGF", i+1)
		var encoded [36]byte
		for j := range name {
			encoded[j] = name[j] ^ 0xCC
		}
		if _, err := list.Write(encoded[:]); err != nil {
			return err
		}
		offset += uint64(count)
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return list.Close()
}
