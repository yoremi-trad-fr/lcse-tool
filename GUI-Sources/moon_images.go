package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

type moonImageFile struct {
	path string
	rel  string
}

func (a *App) convertMoonImageFiles(inputPath, outputDir string) BatchResult {
	info, err := os.Stat(inputPath)
	if err != nil {
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	kind := "files"
	if info.IsDir() {
		kind = "directory"
	}
	return a.MoonConvertTGF("t2p", kind, []string{inputPath}, outputDir)
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
