package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"sync"
)

var (
	testJPEGOnce sync.Once
	testJPEGData []byte
)

// testJPEG is a small stand-in photo, built once and reused.
func testJPEG() []byte {
	testJPEGOnce.Do(func() {
		img := image.NewRGBA(image.Rect(0, 0, 600, 400))

		for y := 0; y < 400; y++ {
			for x := 0; x < 600; x++ {
				img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
			}
		}

		var buf bytes.Buffer
		_ = jpeg.Encode(&buf, img, nil)

		testJPEGData = buf.Bytes()
	})

	return testJPEGData
}
