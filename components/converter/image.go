package converter

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// a4WidthPx is the pixel width of the A4 content area at 96 DPI,
const a4WidthPx = 718

func CenterImageOnCanvas(data []byte) []byte {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}

	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	if srcW >= a4WidthPx {
		return data
	}

	canvas := image.NewRGBA(image.Rect(0, 0, a4WidthPx, srcH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Left-align instead of centering
	destRect := image.Rect(0, 0, srcW, srcH)
	draw.Draw(canvas, destRect, src, src.Bounds().Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return data
	}
	return buf.Bytes()
}
