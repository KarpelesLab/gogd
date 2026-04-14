package gogd

import (
	"bytes"
	"compress/zlib"
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

// --- GD2 ---

const (
	gd2FormatRawPalette        = 1
	gd2FormatCompressedPalette = 2
	gd2FormatRawTrueColor      = 3
	gd2FormatCompressedTC      = 4
)

// ImageCreateFromGD2 decodes a libgd GD2 image from r. Supports both
// truecolor and palette, raw and zlib-compressed chunked variants.
func ImageCreateFromGD2(r io.Reader) (*Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(data) < 18 {
		return nil, errors.New("gogd: GD2 header truncated")
	}
	if string(data[:4]) != "gd2\x00" {
		return nil, fmt.Errorf("gogd: not a GD2 image (magic %q)", string(data[:4]))
	}
	width := int(binary.BigEndian.Uint16(data[6:8]))
	height := int(binary.BigEndian.Uint16(data[8:10]))
	chunkSize := int(binary.BigEndian.Uint16(data[10:12]))
	format := int(binary.BigEndian.Uint16(data[12:14]))
	nChunksX := int(binary.BigEndian.Uint16(data[14:16]))
	nChunksY := int(binary.BigEndian.Uint16(data[16:18]))
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("gogd: GD2 invalid dims %dx%d", width, height)
	}
	compressed := format == gd2FormatCompressedPalette || format == gd2FormatCompressedTC
	truecolor := format == gd2FormatRawTrueColor || format == gd2FormatCompressedTC
	if !compressed && chunkSize != 0 {
		// Raw format doesn't use chunks in any meaningful way; libgd still
		// writes the field but ignores it.
	}
	if compressed && (chunkSize <= 0 || nChunksX <= 0 || nChunksY <= 0) {
		return nil, fmt.Errorf("gogd: GD2 compressed but chunk dims invalid (%d %dx%d)", chunkSize, nChunksX, nChunksY)
	}

	pos := 18
	var palette []color.NRGBA
	var transparent int32
	if truecolor {
		// Truecolor info: 1-byte flag + 4-byte transparent.
		if pos+5 > len(data) {
			return nil, errors.New("gogd: GD2 truecolor header truncated")
		}
		pos++ // skip truecolor flag
		transparent = int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		pos += 4
	} else {
		// Palette info: 1-byte flag + 2-byte ncolors + 4-byte transparent + 256*4 palette.
		if pos+1+2+4+256*4 > len(data) {
			return nil, errors.New("gogd: GD2 palette header truncated")
		}
		pos++ // skip truecolor flag (should be 0)
		nColors := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		transparent = int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		pos += 4
		palette = make([]color.NRGBA, nColors)
		for i := 0; i < 256; i++ {
			entry := data[pos : pos+4]
			if i < nColors {
				palette[i] = color.NRGBA{R: entry[0], G: entry[1], B: entry[2], A: gdAlphaToStdAlpha(int(entry[3]))}
			}
			pos += 4
		}
	}

	// Gather the raw pixel bytes (decompress chunks if necessary).
	bpp := 1
	if truecolor {
		bpp = 4
	}
	pixels := make([]byte, width*height*bpp)

	if compressed {
		// Chunk index table: nChunksX * nChunksY entries of (offset, size) as int32 BE.
		tableBytes := nChunksX * nChunksY * 8
		if pos+tableBytes > len(data) {
			return nil, errors.New("gogd: GD2 chunk index truncated")
		}
		type chunk struct{ offset, size int }
		chunks := make([]chunk, nChunksX*nChunksY)
		for i := range chunks {
			chunks[i].offset = int(binary.BigEndian.Uint32(data[pos : pos+4]))
			chunks[i].size = int(binary.BigEndian.Uint32(data[pos+4 : pos+8]))
			pos += 8
		}
		for cy := 0; cy < nChunksY; cy++ {
			for cx := 0; cx < nChunksX; cx++ {
				ci := cy*nChunksX + cx
				off, sz := chunks[ci].offset, chunks[ci].size
				if off < 0 || sz <= 0 || off+sz > len(data) {
					return nil, fmt.Errorf("gogd: GD2 chunk %d offset/size out of range", ci)
				}
				zr, err := zlib.NewReader(bytes.NewReader(data[off : off+sz]))
				if err != nil {
					return nil, fmt.Errorf("gogd: GD2 chunk %d zlib: %w", ci, err)
				}
				raw, err := io.ReadAll(zr)
				zr.Close()
				if err != nil {
					return nil, fmt.Errorf("gogd: GD2 chunk %d read: %w", ci, err)
				}
				// Paste the chunk into the pixel buffer.
				x0 := cx * chunkSize
				y0 := cy * chunkSize
				cw := chunkSize
				if x0+cw > width {
					cw = width - x0
				}
				ch := chunkSize
				if y0+ch > height {
					ch = height - y0
				}
				for row := 0; row < ch; row++ {
					srcOff := row * chunkSize * bpp
					dstOff := ((y0+row)*width + x0) * bpp
					copy(pixels[dstOff:dstOff+cw*bpp], raw[srcOff:srcOff+cw*bpp])
				}
			}
		}
	} else {
		if pos+len(pixels) > len(data) {
			return nil, errors.New("gogd: GD2 raw pixel data truncated")
		}
		copy(pixels, data[pos:pos+len(pixels)])
	}

	if truecolor {
		img := ImageCreateTrueColor(width, height)
		ImageAlphaBlending(img, false)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				off := (y*width + x) * 4
				img.nrgba.SetNRGBA(x, y, color.NRGBA{
					R: pixels[off+1],
					G: pixels[off+2],
					B: pixels[off+3],
					A: gdAlphaToStdAlpha(int(pixels[off])),
				})
			}
		}
		if transparent >= 0 {
			img.transparent = Color(transparent)
		}
		ImageAlphaBlending(img, true)
		return img, nil
	}

	img := ImageCreate(width, height)
	for _, pc := range palette {
		img.pal.Palette = append(img.pal.Palette, pc)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.pal.SetColorIndex(x, y, pixels[y*width+x])
		}
	}
	if transparent >= 0 {
		img.transparent = Color(transparent)
	}
	return img, nil
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
