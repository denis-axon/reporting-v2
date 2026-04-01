package converter

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// AddTitleToImage draws a title string onto the top of a PNG image.
// It adds a white banner at the top and renders the text in black.
func AddTitleToImage(data []byte, title string) []byte {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}

	const bannerHeight = 28
	const fontSize = 14

	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	// New canvas: same width, extra height at top for the title banner
	dst := image.NewRGBA(image.Rect(0, 0, srcW, srcH+bannerHeight))

	// Fill entire canvas white
	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw original image below the banner
	draw.Draw(dst, image.Rect(0, bannerHeight, srcW, srcH+bannerHeight), src, src.Bounds().Min, draw.Src)

	// Draw title text in the banner, horizontally centered
	col := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
	}
	textWidth := d.MeasureString(title).Ceil()
	d.Dot = fixed.Point26_6{
		X: fixed.I((srcW - textWidth) / 2),
		Y: fixed.I(bannerHeight - 8), // baseline within the banner
	}
	d.DrawString(title)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return data
	}
	return buf.Bytes()
}
