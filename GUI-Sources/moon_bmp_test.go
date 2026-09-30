package main

import (
	"encoding/binary"
	"image/color"
	"testing"
)

func TestDecodeMoonOneBitBMP(t *testing.T) {
	for _, topDown := range []bool{false, true} {
		bmp := make([]byte, 70)
		copy(bmp, "BM")
		binary.LittleEndian.PutUint32(bmp[2:6], uint32(len(bmp)))
		binary.LittleEndian.PutUint32(bmp[10:14], 62)
		binary.LittleEndian.PutUint32(bmp[14:18], 40)
		binary.LittleEndian.PutUint32(bmp[18:22], 2)
		if topDown {
			height := int32(-2)
			binary.LittleEndian.PutUint32(bmp[22:26], uint32(height))
		} else {
			binary.LittleEndian.PutUint32(bmp[22:26], 2)
		}
		binary.LittleEndian.PutUint16(bmp[26:28], 1)
		binary.LittleEndian.PutUint16(bmp[28:30], 1)
		binary.LittleEndian.PutUint32(bmp[34:38], 8)
		binary.LittleEndian.PutUint32(bmp[46:50], 2)
		copy(bmp[58:62], []byte{255, 255, 255, 0})
		if topDown {
			bmp[62], bmp[66] = 0x40, 0x80
		} else {
			bmp[62], bmp[66] = 0x80, 0x40
		}

		img, err := decodeMoonBMP(bmp)
		if err != nil {
			t.Fatalf("topDown=%v: %v", topDown, err)
		}
		if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
			t.Fatalf("topDown=%v: unexpected bounds %v", topDown, img.Bounds())
		}
		want := [2][2]color.RGBA{
			{{0, 0, 0, 255}, {255, 255, 255, 255}},
			{{255, 255, 255, 255}, {0, 0, 0, 255}},
		}
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				if got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA); got != want[y][x] {
					t.Fatalf("topDown=%v pixel (%d,%d): got %v, want %v", topDown, x, y, got, want[y][x])
				}
			}
		}
	}
}
