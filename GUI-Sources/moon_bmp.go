package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"

	"golang.org/x/image/bmp"
)

func decodeMoonBMP(data []byte) (image.Image, error) {
	if len(data) >= 54 && bytes.Equal(data[:2], []byte{'B', 'M'}) && binary.LittleEndian.Uint16(data[28:30]) == 1 {
		return decodeMoonOneBitBMP(data)
	}
	return bmp.Decode(bytes.NewReader(data))
}

// x/image/bmp does not decode the uncompressed 1-bit BMP produced from
// FG165.TGF by the original MOON extractor.
func decodeMoonOneBitBMP(data []byte) (image.Image, error) {
	if len(data) < 62 || binary.LittleEndian.Uint32(data[14:18]) < 40 {
		return nil, fmt.Errorf("en-tete BMP 1 bit invalide")
	}
	width := int64(int32(binary.LittleEndian.Uint32(data[18:22])))
	signedHeight := int64(int32(binary.LittleEndian.Uint32(data[22:26])))
	if width <= 0 || signedHeight == 0 || binary.LittleEndian.Uint16(data[26:28]) != 1 || binary.LittleEndian.Uint32(data[30:34]) != 0 {
		return nil, fmt.Errorf("dimensions ou compression BMP 1 bit invalides")
	}
	height := signedHeight
	if height < 0 {
		height = -height
	}
	if width*height > 100_000_000 {
		return nil, fmt.Errorf("BMP 1 bit trop grand")
	}
	paletteOffset := int64(14) + int64(binary.LittleEndian.Uint32(data[14:18]))
	pixelOffset := int64(binary.LittleEndian.Uint32(data[10:14]))
	stride := ((width + 31) / 32) * 4
	if paletteOffset < 54 || paletteOffset+8 > pixelOffset || pixelOffset+stride*height > int64(len(data)) {
		return nil, fmt.Errorf("palette ou pixels BMP 1 bit incomplets")
	}
	palette := make(color.Palette, 2)
	for i := 0; i < 2; i++ {
		o := int(paletteOffset) + 4*i
		palette[i] = color.RGBA{R: data[o+2], G: data[o+1], B: data[o], A: 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, int(width), int(height)), palette)
	for y := int64(0); y < height; y++ {
		fileRow := y
		if signedHeight > 0 {
			fileRow = height - 1 - y
		}
		row := pixelOffset + fileRow*stride
		for x := int64(0); x < width; x++ {
			bit := (data[int(row+x/8)] >> uint(7-x%8)) & 1
			img.Pix[int(y)*img.Stride+int(x)] = bit
		}
	}
	return img, nil
}
