package gogd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	"io"
)

// GD format v1 layout:
//
//	magic  2B  0xFF 0xFE (palette) | 0xFF 0xFF (truecolor)
//	width  2B  big-endian uint16
//	height 2B  big-endian uint16
//	truecolor flag 1B  (1 = truecolor, 0 = palette)
//	palette only:
//	  ncolors   2B  big-endian uint16
//	  transparent index  4B  big-endian int32 (-1 = none)
//	  palette entries:  ncolors * 4B (r, g, b, unused)
//	  pixels: width*height bytes (palette index each)
//	truecolor only:
//	  transparent color 4B  big-endian int32 (packed truecolor or -1)
//	  pixels: width*height*4 bytes (each pixel: a, r, g, b)
//
// Note: the "truecolor flag" actually duplicates information already
// encoded in the magic. libgd sets magic 0xFFFE for palette images (v1)
// and 0xFFFF for truecolor. We accept either convention on read.

// ImageCreateFromGD decodes a libgd v1 "GD" image from r.
func ImageCreateFromGD(r io.Reader) (*Image, error) {
	var magic [2]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, err
	}
	if magic[0] != 0xFF || (magic[1] != 0xFE && magic[1] != 0xFF) {
		return nil, fmt.Errorf("gogd: not a GD1 image (magic %x %x)", magic[0], magic[1])
	}
	var width, height uint16
	if err := binary.Read(r, binary.BigEndian, &width); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &height); err != nil {
		return nil, err
	}
	var flag uint8
	if err := binary.Read(r, binary.BigEndian, &flag); err != nil {
		return nil, err
	}
	if flag == 1 {
		return readGDTrueColor(r, int(width), int(height))
	}
	return readGDPalette(r, int(width), int(height))
}

func readGDTrueColor(r io.Reader, w, h int) (*Image, error) {
	var trans int32
	if err := binary.Read(r, binary.BigEndian, &trans); err != nil {
		return nil, err
	}
	pixels := make([]byte, w*h*4)
	if _, err := io.ReadFull(r, pixels); err != nil {
		return nil, err
	}
	img := ImageCreateTrueColor(w, h)
	ImageAlphaBlending(img, false)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			a := pixels[off]
			r8 := pixels[off+1]
			g8 := pixels[off+2]
			b8 := pixels[off+3]
			img.nrgba.SetNRGBA(x, y, color.NRGBA{
				R: r8,
				G: g8,
				B: b8,
				A: gdAlphaToStdAlpha(int(a)),
			})
		}
	}
	if trans >= 0 {
		img.transparent = Color(trans)
	}
	ImageAlphaBlending(img, true)
	return img, nil
}

func readGDPalette(r io.Reader, w, h int) (*Image, error) {
	var nColors uint16
	if err := binary.Read(r, binary.BigEndian, &nColors); err != nil {
		return nil, err
	}
	if nColors == 0 || nColors > 256 {
		return nil, fmt.Errorf("gogd: invalid palette size %d", nColors)
	}
	var trans int32
	if err := binary.Read(r, binary.BigEndian, &trans); err != nil {
		return nil, err
	}
	palBytes := make([]byte, int(nColors)*4)
	if _, err := io.ReadFull(r, palBytes); err != nil {
		return nil, err
	}
	img := ImageCreate(w, h)
	for i := 0; i < int(nColors); i++ {
		off := i * 4
		ImageColorAllocate(img, int(palBytes[off]), int(palBytes[off+1]), int(palBytes[off+2]))
	}
	pixels := make([]byte, w*h)
	if _, err := io.ReadFull(r, pixels); err != nil {
		return nil, err
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := pixels[y*w+x]
			if int(idx) < len(img.pal.Palette) {
				img.pal.SetColorIndex(x, y, idx)
			}
		}
	}
	if trans >= 0 {
		img.transparent = Color(trans)
	}
	return img, nil
}

// ImageGD writes img to w in libgd's v1 GD format. Palette images emit
// the palette variant; truecolor images emit the truecolor variant.
func ImageGD(img *Image, w io.Writer) error {
	if img == nil {
		return errNilImage
	}
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width <= 0 || height <= 0 || width > 0xFFFF || height > 0xFFFF {
		return fmt.Errorf("gogd: GD image dims %dx%d out of range", width, height)
	}
	if img.nrgba != nil {
		return writeGDTrueColor(w, img, width, height)
	}
	if img.pal != nil {
		return writeGDPalette(w, img, width, height)
	}
	return errors.New("gogd: GD cannot encode generic image types")
}

func writeGDTrueColor(w io.Writer, img *Image, width, height int) error {
	// Magic for truecolor v1: 0xFF 0xFF.
	if _, err := w.Write([]byte{0xFF, 0xFF}); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(width)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(height)); err != nil {
		return err
	}
	if _, err := w.Write([]byte{1}); err != nil {
		return err
	}
	trans := int32(-1)
	if img.transparent != ColorNone && img.transparent >= 0 {
		trans = int32(img.transparent)
	}
	if err := binary.Write(w, binary.BigEndian, trans); err != nil {
		return err
	}
	buf := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			off := (y*width + x) * 4
			c := img.nrgba.NRGBAAt(x, y)
			buf[off] = byte(stdAlphaToGD(c.A))
			buf[off+1] = c.R
			buf[off+2] = c.G
			buf[off+3] = c.B
		}
	}
	_, err := w.Write(buf)
	return err
}

func writeGDPalette(w io.Writer, img *Image, width, height int) error {
	// Magic for palette v1: 0xFF 0xFE.
	if _, err := w.Write([]byte{0xFF, 0xFE}); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(width)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint16(height)); err != nil {
		return err
	}
	if _, err := w.Write([]byte{0}); err != nil {
		return err
	}
	nColors := len(img.pal.Palette)
	if nColors > 256 {
		nColors = 256
	}
	if err := binary.Write(w, binary.BigEndian, uint16(nColors)); err != nil {
		return err
	}
	trans := int32(-1)
	if img.transparent != ColorNone && img.transparent >= 0 {
		trans = int32(img.transparent)
	}
	if err := binary.Write(w, binary.BigEndian, trans); err != nil {
		return err
	}
	palBuf := make([]byte, nColors*4)
	for i := 0; i < nColors; i++ {
		nc := color.NRGBAModel.Convert(img.pal.Palette[i]).(color.NRGBA)
		palBuf[i*4] = nc.R
		palBuf[i*4+1] = nc.G
		palBuf[i*4+2] = nc.B
	}
	if _, err := w.Write(palBuf); err != nil {
		return err
	}
	pixBuf := make([]byte, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixBuf[y*width+x] = img.pal.ColorIndexAt(x, y)
		}
	}
	_, err := w.Write(pixBuf)
	return err
}
