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
	"strings"
	"testing"
)

func samePixels(t *testing.T, a, b image.Image) {
	t.Helper()
	if a.Bounds() != b.Bounds() {
		t.Fatalf("dimensions %v != %v", a.Bounds(), b.Bounds())
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if color.NRGBAModel.Convert(a.At(x, y)) != color.NRGBAModel.Convert(b.At(x, y)) {
				t.Fatalf("pixel different (%d,%d)", x, y)
			}
		}
	}
}

func TestMoonTGFRoundTripPaddingRunsAndTails(t *testing.T) {
	for _, w := range []int{1, 2, 3, 4, 5, 120, 160, 640} {
		img := image.NewNRGBA(image.Rect(0, 0, w, 4))
		for y := 0; y < 4; y++ {
			for x := 0; x < w; x++ {
				img.SetNRGBA(x, y, color.NRGBA{R: byte(x * 31), G: byte(y * 73), B: byte(x + y), A: 255})
			}
		}
		bmp, err := encodeMoonBMP(img)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeMoonTGF(encodeMoonTGF(bmp))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bmp, decoded) {
			t.Fatalf("width %d: BMP altered", w)
		}
		back, err := decodeMoonBMP(decoded)
		if err != nil {
			t.Fatal(err)
		}
		samePixels(t, img, back)
	}
	for _, n := range []int{255, 256, 1000} {
		data := append([]byte("BM"), bytes.Repeat([]byte{1, 2, 3}, n)...)
		back, err := decodeMoonTGF(encodeMoonTGF(data))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, back) {
			t.Fatal("long run altered")
		}
	}
}

func TestMoonTGFRejectsCorruptDataAndPartialAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 100, A: 128})
	if _, err := encodeMoonBMP(img); err == nil {
		t.Fatal("partial alpha accepted")
	}
	img.SetNRGBA(1, 0, color.NRGBA{})
	bmp, err := encodeMoonBMP(img)
	if err != nil {
		t.Fatal(err)
	}
	back, err := decodeMoonBMP(bmp)
	if err != nil {
		t.Fatal(err)
	}
	if color.NRGBAModel.Convert(back.At(1, 0)) != (color.NRGBA{R: 255, B: 255, A: 255}) {
		t.Fatal("transparent colour key lost")
	}
	good := encodeMoonTGF(bmp)
	for _, bad := range [][]byte{good[:7], good[:len(good)-1], append(append([]byte{}, good...), 255, 1, 2, 3)} {
		if _, err := decodeMoonTGF(bad); err == nil {
			t.Fatal("corrupt stream accepted")
		}
	}
	huge := append([]byte{}, good...)
	binary.LittleEndian.PutUint32(huge, 0xFFFFFFFF)
	if _, err := decodeMoonTGF(huge); err == nil {
		t.Fatal("unbounded allocation accepted")
	}
}

func TestMoonTGFSelectionAndOverwriteSafety(t *testing.T) {
	source, out := t.TempDir(), t.TempDir()
	inputs := []string{}
	for _, name := range []string{"FIRST.png", "SECOND.PNG"} {
		p := filepath.Join(source, name)
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rect(0, 0, 5, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 5; x++ {
				img.Set(x, y, color.RGBA{R: 60, A: 255})
			}
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
		inputs = append(inputs, p)
	}
	app := NewApp()
	if app.MoonConvertTGF("p2t", "archive", inputs, out).Status != "ERROR" {
		t.Fatal("PNG archive fallback accepted")
	}
	if app.MoonConvertTGF("t2p", "files", inputs, out).Status != "ERROR" {
		t.Fatal("wrong direction accepted")
	}
	if app.MoonConvertImages(inputs[0], out).Status != "ERROR" {
		t.Fatal("legacy PNG archive fallback accepted")
	}
	if app.MoonConvertTGF("p2t", "files", inputs, out).Status != "OK" {
		t.Fatal("PNG batch failed")
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 2 {
		t.Fatal("unexpected batch outputs")
	}
	before, _ := os.ReadFile(filepath.Join(out, "FIRST.tgf"))
	if app.MoonConvertTGF("p2t", "files", inputs, out).Status != "ERROR" {
		t.Fatal("overwrite accepted")
	}
	after, _ := os.ReadFile(filepath.Join(out, "FIRST.tgf"))
	if !bytes.Equal(before, after) {
		t.Fatal("existing output changed")
	}
	if err := os.WriteFile(filepath.Join(out, "PLAIN.bmp"), []byte("ignored"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := app.MoonConvertTGF("t2p", "directory", []string{out}, t.TempDir()); got.Status != "OK" || !strings.HasPrefix(got.Detail, "2 PNG") {
		t.Fatalf("BMP wasn't excluded: %+v", got)
	}
}

func TestMoonTGFKnownArchiveOfficialIntegration(t *testing.T) {
	source, extractor := os.Getenv("LCSE_TEST_TGF_SOURCE"), os.Getenv("LCSE_TEST_TGF_EXTRACTOR")
	if source == "" || extractor == "" {
		t.Skip("set TGF source and official extractor")
	}
	paths, err := filepath.Glob(filepath.Join(source, "*.tgf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 479 {
		t.Fatalf("want 479 reference TGF, got %d", len(paths))
	}
	files := []moonImageFile{}
	for _, p := range paths {
		files = append(files, moonImageFile{path: p, rel: filepath.Base(p)})
	}
	dir := t.TempDir()
	if err := writeMoonTGFArchive(dir, files); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	if err := app.runExecutable(extractor, dir); err != nil {
		t.Fatal(err)
	}
	for i, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeMoonTGF(data)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		official, err := os.ReadFile(filepath.Join(dir, "TGF", fmtImageName(i)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decoded, official) {
			t.Fatalf("official bytes differ: %s", p)
		}
		if _, err := decodeMoonBMP(decoded); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
	t.Logf("479 TGF: byte-identical to official BMPs")
}

func TestMoonPNGOfficialIntegration(t *testing.T) {
	source, out, extractor := os.Getenv("LCSE_TEST_PNG_SOURCE"), os.Getenv("LCSE_TEST_PNG_OUTPUT"), os.Getenv("LCSE_TEST_TGF_EXTRACTOR")
	if source == "" || out == "" || extractor == "" {
		t.Skip("set PNG source, output and official extractor")
	}
	inputs, err := filepath.Glob(filepath.Join(source, "*.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 2 {
		t.Fatalf("want exactly two edited PNGs, got %d", len(inputs))
	}
	app := NewApp()
	if got := app.MoonConvertTGF("p2t", "files", inputs, out); got.Status != "OK" {
		t.Fatal(got)
	}
	files := []moonImageFile{}
	for _, p := range inputs {
		files = append(files, moonImageFile{path: filepath.Join(out, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))+".tgf")})
	}
	dir := t.TempDir()
	if err := writeMoonTGFArchive(dir, files); err != nil {
		t.Fatal(err)
	}
	if err := app.runExecutable(extractor, dir); err != nil {
		t.Fatal(err)
	}
	for i, p := range inputs {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		original, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		data, err = os.ReadFile(filepath.Join(dir, "TGF", fmtImageName(i)))
		if err != nil {
			t.Fatal(err)
		}
		back, err := decodeMoonBMP(data)
		if err != nil {
			t.Fatal(err)
		}
		samePixels(t, original, back)
	}
	t.Log("2 edited PNGs -> TGF -> official extractor: identical pixels")
}

func fmtImageName(i int) string { return fmt.Sprintf("IMG%06d.BMP", i+1) }
