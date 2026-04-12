package gogd

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// ImageFontWidth returns the per-character width of the built-in font
// identified by fontID (1..5), matching the sizes reported by PHP gd.
// gogd currently renders every built-in font using a single bitmap face,
// but these dimensions are still useful for layout.
func ImageFontWidth(fontID int) int {
	switch fontID {
	case 1:
		return 5
	case 2:
		return 6
	case 4:
		return 8
	case 5:
		return 9
	}
	return 7
}

// ImageFontHeight returns the per-character height for the built-in font.
func ImageFontHeight(fontID int) int {
	switch fontID {
	case 1:
		return 8
	case 2:
		return 12
	case 4:
		return 16
	case 5:
		return 15
	}
	return 13
}

// ImageLoadFont is a stub for compatibility with PHP code. gd's own .gd
// font-file format is not yet supported; the function always returns 0.
func ImageLoadFont(path string) int {
	return 0
}

// ImageString draws a horizontal string at (x, y). (x, y) is the top-left
// corner of the first character. Accepts any [draw.Image].
func ImageString(dst draw.Image, fontID, x, y int, s string, c Color) bool {
	if dst == nil {
		return false
	}
	return drawBitmapString(dst, basicfont.Face7x13, x, y, s, c)
}

// ImageChar draws a single character at (x, y).
func ImageChar(dst draw.Image, fontID, x, y int, ch string, c Color) bool {
	if ch == "" {
		return false
	}
	return ImageString(dst, fontID, x, y, ch[:1], c)
}

// ImageStringUp draws a string rotated 90° counter-clockwise at (x, y).
// Accepts any [draw.Image].
func ImageStringUp(dst draw.Image, fontID, x, y int, s string, c Color) bool {
	if dst == nil || s == "" {
		return false
	}
	fontH := ImageFontHeight(fontID)
	face := basicfont.Face7x13
	adv, _ := font.BoundString(face, s)
	w := (adv.Max.X - adv.Min.X).Ceil()
	if w <= 0 {
		w = ImageFontWidth(fontID) * len(s)
	}
	h := fontH

	tmp := ImageCreateTrueColor(w, h)
	ImageAlphaBlending(tmp, false)
	transparent := ImageColorAllocateAlpha(tmp, 0, 0, 0, AlphaTransparent)
	ImageFilledRectangle(tmp, 0, 0, w-1, h-1, transparent)
	drawBitmapString(tmp, face, 0, 0, s, c)
	rotated := ImageRotate(tmp, 90, transparent)
	ImageCopy(dst, rotated, x, y-rotated.Height()+1, 0, 0, rotated.Width(), rotated.Height())
	return true
}

// ImageCharUp draws a single character rotated 90° counter-clockwise.
func ImageCharUp(dst draw.Image, fontID, x, y int, ch string, c Color) bool {
	if ch == "" {
		return false
	}
	return ImageStringUp(dst, fontID, x, y, ch[:1], c)
}

// --- TTF text ---

// ImageTTFText draws TTF text at the given baseline position and returns
// the bounding box of the drawn text as 8 integers: the (x, y) pairs for
// the lower-left, lower-right, upper-right, and upper-left corners, in
// that order (matching PHP imagettftext). Accepts any [draw.Image].
func ImageTTFText(img draw.Image, size, angle float64, x, y int, c Color, fontPath, text string) ([8]int, error) {
	face, err := loadTTFFace(fontPath, size)
	if err != nil {
		return [8]int{}, err
	}
	defer face.Close()
	return drawOrMeasureTTF(img, face, angle, x, y, c, text, true)
}

// ImageTTFBBox returns the bounding box for the given TTF text without
// rendering. The returned slice matches [ImageTTFText].
func ImageTTFBBox(size, angle float64, fontPath, text string) ([8]int, error) {
	face, err := loadTTFFace(fontPath, size)
	if err != nil {
		return [8]int{}, err
	}
	defer face.Close()
	return drawOrMeasureTTF(nil, face, angle, 0, 0, 0, text, false)
}

// ImageFTText is an alias for [ImageTTFText] (FreeType shim).
func ImageFTText(img draw.Image, size, angle float64, x, y int, c Color, fontPath, text string) ([8]int, error) {
	return ImageTTFText(img, size, angle, x, y, c, fontPath, text)
}

// ImageFTBBox is an alias for [ImageTTFBBox].
func ImageFTBBox(size, angle float64, fontPath, text string) ([8]int, error) {
	return ImageTTFBBox(size, angle, fontPath, text)
}

// --- internals ---

func drawBitmapString(dst draw.Image, face font.Face, x, y int, s string, c Color) bool {
	ascent := face.Metrics().Ascent.Ceil()
	target := dst
	if g, ok := dst.(*Image); ok && g != nil && g.nrgba != nil {
		target = g.nrgba
	}
	d := &font.Drawer{
		Dst:  target,
		Src:  image.NewUniform(gdColorToNRGBA(c)),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y + ascent),
		},
	}
	d.DrawString(s)
	return true
}

func loadTTFFace(path string, size float64) (font.Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gogd: read font %q: %w", path, err)
	}
	fnt, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("gogd: parse font %q: %w", path, err)
	}
	return opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    size,
		DPI:     96,
		Hinting: font.HintingFull,
	})
}

func drawOrMeasureTTF(img draw.Image, face font.Face, angle float64, x, y int, c Color, text string, drawText bool) ([8]int, error) {
	b, _ := font.BoundString(face, text)
	bXmin, bYmin := b.Min.X.Floor(), b.Min.Y.Floor()
	bXmax, bYmax := b.Max.X.Ceil(), b.Max.Y.Ceil()

	corners := [4][2]float64{
		{float64(bXmin), float64(bYmax)}, // lower left
		{float64(bXmax), float64(bYmax)}, // lower right
		{float64(bXmax), float64(bYmin)}, // upper right
		{float64(bXmin), float64(bYmin)}, // upper left
	}

	if angle != 0 {
		// PHP rotates text counter-clockwise around (x, y); the bounding
		// box follows the same rotation.
		rad := angle * math.Pi / 180
		cos, sin := math.Cos(rad), math.Sin(rad)
		for i := range corners {
			px, py := corners[i][0], corners[i][1]
			// Image coords (Y-down): visual CCW matches math CW.
			corners[i][0] = px*cos + py*sin
			corners[i][1] = -px*sin + py*cos
		}
	}

	var out [8]int
	for i, p := range corners {
		out[2*i] = x + int(math.Round(p[0]))
		out[2*i+1] = y + int(math.Round(p[1]))
	}

	if drawText && img != nil {
		target := img
		if g, ok := img.(*Image); ok && g != nil && g.nrgba != nil {
			target = g.nrgba
		}
		if angle == 0 {
			d := &font.Drawer{
				Dst:  target,
				Src:  image.NewUniform(gdColorToNRGBA(c)),
				Face: face,
				Dot:  fixed.P(x, y),
			}
			d.DrawString(text)
		} else {
			// Render to a temp image then rotate + paste.
			w := bXmax - bXmin + 1
			h := bYmax - bYmin + 1
			if w < 1 {
				w = 1
			}
			if h < 1 {
				h = 1
			}
			tmp := ImageCreateTrueColor(w, h)
			ImageAlphaBlending(tmp, false)
			transparent := ImageColorAllocateAlpha(tmp, 0, 0, 0, AlphaTransparent)
			ImageFilledRectangle(tmp, 0, 0, w-1, h-1, transparent)
			dd := &font.Drawer{
				Dst:  tmp.nrgba,
				Src:  image.NewUniform(gdColorToNRGBA(c)),
				Face: face,
				Dot:  fixed.P(-bXmin, -bYmin),
			}
			dd.DrawString(text)
			rotated := ImageRotate(tmp, angle, transparent)
			// Align the lower-left corner of the rotated bbox back to (x+bXmin, y+bYmax).
			rad := angle * math.Pi / 180
			cos, sin := math.Cos(rad), math.Sin(rad)
			origLLx := float64(bXmin)
			origLLy := float64(bYmax)
			rotLLx := origLLx*cos + origLLy*sin
			rotLLy := -origLLx*sin + origLLy*cos
			// Rotated image has its own bounds; the rotation in ImageRotate
			// centres on the rotated box. Paste at an offset that puts the
			// lower-left of the text at (x + bXmin, y + bYmax) in output.
			cx := int(math.Round(float64(x) + rotLLx - float64(rotated.Width())/2 + float64(w)/2))
			cy := int(math.Round(float64(y) + rotLLy - float64(rotated.Height())/2 + float64(h)/2))
			ImageCopy(img, rotated, cx, cy, 0, 0, rotated.Width(), rotated.Height())
		}
	}

	return out, nil
}
