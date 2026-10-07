package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxMoonImageBytes = 256 * 1024 * 1024

// MOON Kit's decoder (moon_extractTGF.exe, 0x4012d0): an unpacked BMP
// length, a repeated-unit length, then RLE. 0 copies a final byte tail;
// 1 copies N units; 2..255 repeat one unit. The BMP includes row padding.
func decodeMoonTGF(data []byte) ([]byte, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("en-tete TGF incomplet")
	}
	size := int64(binary.LittleEndian.Uint32(data))
	unit := int64(binary.LittleEndian.Uint32(data[4:]))
	if size < 54 || size > maxMoonImageBytes || unit < 1 || unit > 64 {
		return nil, fmt.Errorf("taille ou unite TGF invalide")
	}
	out := make([]byte, 0, int(size))
	for i := 8; i < len(data); {
		token := data[i]
		i++
		count, inputBytes := int64(token), unit
		if token <= 1 {
			if i == len(data) || data[i] == 0 {
				return nil, fmt.Errorf("bloc TGF incomplet ou vide")
			}
			count = int64(data[i])
			i++
			inputBytes = count
			if token == 1 {
				inputBytes *= unit
			}
			count = 1
		}
		if int64(i)+inputBytes > int64(len(data)) || int64(len(out))+inputBytes*count > size {
			return nil, fmt.Errorf("bloc TGF hors limites")
		}
		for n := int64(0); n < count; n++ {
			out = append(out, data[i:i+int(inputBytes)]...)
		}
		i += int(inputBytes)
		if token == 0 && i != len(data) {
			return nil, fmt.Errorf("queue TGF avant la fin du flux")
		}
	}
	if int64(len(out)) != size || !bytes.Equal(out[:2], []byte("BM")) {
		return nil, fmt.Errorf("BMP TGF incomplet ou signature invalide")
	}
	return out, nil
}

func encodeMoonTGF(bmp []byte) []byte {
	const unit = 3
	out := make([]byte, 8, len(bmp)+len(bmp)/255+16)
	binary.LittleEndian.PutUint32(out, uint32(len(bmp)))
	binary.LittleEndian.PutUint32(out[4:], unit)
	end := len(bmp) - len(bmp)%unit
	run := func(pos int) int {
		n := 1
		for n < 255 && pos+(n+1)*unit <= end && bytes.Equal(bmp[pos:pos+unit], bmp[pos+n*unit:pos+(n+1)*unit]) {
			n++
		}
		return n
	}
	for i := 0; i < end; {
		if n := run(i); n >= 2 {
			out = append(out, byte(n))
			out = append(out, bmp[i:i+unit]...)
			i += n * unit
		} else {
			start := i
			i += unit
			for i < end && (i-start)/unit < 255 && run(i) == 1 {
				i += unit
			}
			out = append(out, 1, byte((i-start)/unit))
			out = append(out, bmp[start:i]...)
		}
	}
	if end < len(bmp) {
		out = append(out, 0, byte(len(bmp)-end))
		out = append(out, bmp[end:]...)
	}
	return out
}

// A 24-bit, bottom-up BMP, with four-byte row alignment. MOON uses
// magenta as its colour key; full transparency becomes magenta, while
// partial transparency is rejected rather than silently flattened.
func encodeMoonBMP(img image.Image) ([]byte, error) {
	b := img.Bounds()
	w, h := int64(b.Dx()), int64(b.Dy())
	stride := (w*3 + 3) &^ 3
	if w < 1 || h < 1 || w > 32768 || h > 32768 || 54+stride*h > maxMoonImageBytes {
		return nil, fmt.Errorf("dimensions PNG trop grandes ou invalides")
	}
	out := make([]byte, 54+int(stride*h))
	copy(out, "BM")
	binary.LittleEndian.PutUint32(out[2:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[10:], 54)
	binary.LittleEndian.PutUint32(out[14:], 40)
	binary.LittleEndian.PutUint32(out[18:], uint32(w))
	binary.LittleEndian.PutUint32(out[22:], uint32(h))
	binary.LittleEndian.PutUint16(out[26:], 1)
	binary.LittleEndian.PutUint16(out[28:], 24)
	binary.LittleEndian.PutUint32(out[34:], uint32(stride*h))
	for y := 0; y < int(h); y++ {
		for x := 0; x < int(w); x++ {
			c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			if c.A == 0 {
				c = color.NRGBA{R: 255, B: 255, A: 255}
			} else if c.A != 255 {
				return nil, fmt.Errorf("transparence partielle au pixel (%d, %d) : TGF accepte seulement des pixels opaques ou transparents", x, y)
			}
			o := 54 + (int(h)-1-y)*int(stride) + x*3
			out[o], out[o+1], out[o+2] = c.B, c.G, c.R
		}
	}
	return out, nil
}

func convertMoonTGFFile(source, target, direction string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.Size() > maxMoonImageBytes {
		return fmt.Errorf("fichier image trop grand")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	var output []byte
	if direction == "p2t" {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("PNG invalide : %w", err)
		}
		if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 32768 || cfg.Height > 32768 || int64(cfg.Width)*int64(cfg.Height)*4 > maxMoonImageBytes {
			return fmt.Errorf("dimensions PNG trop grandes ou invalides")
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return err
		}
		bmp, err := encodeMoonBMP(img)
		if err != nil {
			return err
		}
		output = encodeMoonTGF(bmp)
	} else {
		bmp, err := decodeMoonTGF(data)
		if err != nil {
			return err
		}
		img, err := decodeMoonBMP(bmp)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		output = buf.Bytes()
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(output)
	closeErr := f.Close()
	if writeErr != nil {
		os.Remove(target)
		return writeErr
	}
	if closeErr != nil {
		os.Remove(target)
		return closeErr
	}
	return nil
}

func (a *App) SelectImageFiles(direction string) []string {
	return a.selectImageFiles(direction)
}

// Source kind and direction are explicit. A PNG can never fall back to
// archive extraction. Batch validation completes before writing any file.
func (a *App) MoonConvertTGF(direction, sourceKind string, inputs []string, outputDir string) BatchResult {
	fail := func(err error) BatchResult {
		a.logError(err.Error())
		return BatchResult{Status: "ERROR", Detail: err.Error()}
	}
	if (direction != "p2t" && direction != "t2p") || len(inputs) == 0 || outputDir == "" {
		return fail(fmt.Errorf("sens, source et dossier de sortie requis"))
	}
	if sourceKind == "archive" {
		if direction != "t2p" || len(inputs) != 1 {
			return fail(fmt.Errorf("une archive est possible uniquement pour TGF -> PNG"))
		}
		return a.MoonExtractImagesFromArchive(inputs[0], outputDir)
	}
	ext, outputExt := ".tgf", ".png"
	if direction == "p2t" {
		ext, outputExt = ".png", ".tgf"
	}
	files := []moonImageFile{}
	switch sourceKind {
	case "files":
		for _, p := range inputs {
			files = append(files, moonImageFile{path: p, rel: filepath.Base(p)})
		}
	case "directory":
		if len(inputs) != 1 {
			return fail(fmt.Errorf("choisir un seul dossier"))
		}
		info, err := os.Stat(inputs[0])
		if err != nil {
			return fail(err)
		}
		if !info.IsDir() {
			return fail(fmt.Errorf("la source doit etre un dossier"))
		}
		err = filepath.WalkDir(inputs[0], func(p string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.EqualFold(filepath.Ext(p), ext) {
				rel, err := filepath.Rel(inputs[0], p)
				if err != nil {
					return err
				}
				files = append(files, moonImageFile{path: p, rel: rel})
			}
			return nil
		})
		if err != nil {
			return fail(err)
		}
	default:
		return fail(fmt.Errorf("source inconnue : choisir Fichiers, Dossier ou Archive"))
	}
	if len(files) == 0 {
		return fail(fmt.Errorf("aucun fichier %s dans la source", ext))
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	seen := map[string]bool{}
	for i := range files {
		file := &files[i]
		if !strings.EqualFold(filepath.Ext(file.path), ext) {
			return fail(fmt.Errorf("%s : fichier %s requis pour ce sens", file.path, ext))
		}
		info, err := os.Stat(file.path)
		if err != nil {
			return fail(err)
		}
		if !info.Mode().IsRegular() {
			return fail(fmt.Errorf("%s : fichier ordinaire requis", file.path))
		}
		file.rel = strings.TrimSuffix(file.rel, filepath.Ext(file.rel)) + outputExt
		key := strings.ToLower(file.rel)
		if seen[key] {
			return fail(fmt.Errorf("deux sources ont le meme nom de sortie : %s", file.rel))
		}
		seen[key] = true
		target := filepath.Join(outputDir, file.rel)
		if _, err := os.Lstat(target); err == nil {
			return fail(fmt.Errorf("%s existe deja : choisir un dossier de sortie vide", target))
		} else if !os.IsNotExist(err) {
			return fail(err)
		}
	}
	a.logSection(strings.ToUpper(strings.TrimPrefix(ext, ".")) + " -> " + strings.ToUpper(strings.TrimPrefix(outputExt, ".")))
	converted, failures := 0, 0
	for _, file := range files {
		if err := convertMoonTGFFile(file.path, filepath.Join(outputDir, file.rel), direction); err != nil {
			a.logError(fmt.Sprintf("%s : %v", file.rel, err))
			failures++
		} else {
			converted++
			if converted <= 25 || converted%100 == 0 {
				a.log(fmt.Sprintf("[%d] %s", converted, file.rel))
			}
		}
	}
	msg := fmt.Sprintf("%d %s crees, %d erreurs", converted, strings.ToUpper(strings.TrimPrefix(outputExt, ".")), failures)
	if failures > 0 {
		return fail(fmt.Errorf("%s", msg))
	}
	a.logOK(msg)
	return BatchResult{Status: "OK", Detail: msg}
}
