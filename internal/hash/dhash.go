package hash

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func DHash64(imgBytes []byte) (uint64, error) {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return 0, fmt.Errorf("dhash: decode: %w", err)
	}
	return DHash64FromImage(img), nil
}

func DHash64FromImage(img image.Image) uint64 {
	dst := image.NewRGBA(image.Rect(0, 0, 9, 8))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	var h uint64
	var bit uint = 0
	for y := 0; y < 8; y++ {
		var row [9]uint8
		for x := 0; x < 9; x++ {
			row[x] = luma8(dst.At(x, y))
		}
		for x := 0; x < 8; x++ {
			if row[x] > row[x+1] {
				h |= 1 << bit
			}
			bit++
		}
	}
	return h
}

func luma8(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()
	R := uint32(r >> 8)
	G := uint32(g >> 8)
	B := uint32(b >> 8)
	Y := (299*R + 587*G + 114*B + 500) / 1000
	return uint8(Y)
}
