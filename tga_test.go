package gogd

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// writeTestTGA emits a minimal uncompressed 32-bit truecolor TGA
// (bottom-left origin) from a *gogd.Image.
func writeTestTGA(t *testing.T, img *Image) []byte {
	t.Helper()
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var buf bytes.Buffer
	hdr := make([]byte, 18)
	hdr[2] = 2 // uncompressed truecolor
	binary.LittleEndian.PutUint16(hdr[12:14], uint16(w))
	binary.LittleEndian.PutUint16(hdr[14:16], uint16(h))
	hdr[16] = 32 // bits per pixel
	hdr[17] = 8  // 8 alpha bits, bottom-left origin
	buf.Write(hdr)
	// TGA stores rows bottom-up when topDown bit is unset.
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := img.nrgba.NRGBAAt(x, y)
			buf.WriteByte(c.B)
			buf.WriteByte(c.G)
			buf.WriteByte(c.R)
			buf.WriteByte(c.A)
		}
	}
	return buf.Bytes()
}

// writeTestTGA_RLE emits an RLE-compressed 24-bit truecolor TGA.
func writeTestTGA_RLE(t *testing.T, img *Image) []byte {
	t.Helper()
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var buf bytes.Buffer
	hdr := make([]byte, 18)
	hdr[2] = 10 // RLE truecolor
	binary.LittleEndian.PutUint16(hdr[12:14], uint16(w))
	binary.LittleEndian.PutUint16(hdr[14:16], uint16(h))
	hdr[16] = 24
	hdr[17] = 0
	buf.Write(hdr)
	// Serialize pixels into BGR, bottom-up.
	rows := make([]byte, 0, w*h*3)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			c := img.nrgba.NRGBAAt(x, y)
			rows = append(rows, c.B, c.G, c.R)
		}
	}
	// Simple RLE: emit each pixel as a raw packet of 1 (count byte 0x00 + one pixel).
	// This isn't compact but is always valid.
	for i := 0; i < len(rows); i += 3 {
		buf.WriteByte(0) // raw packet, count = 1
		buf.Write(rows[i : i+3])
	}
	return buf.Bytes()
}

func TestTGAUncompressedRoundtrip(t *testing.T) {
	src := ImageCreateTrueColor(4, 3)
	ImageAlphaBlending(src, false)
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			c := ImageColorAllocateAlpha(src, x*60, y*80, 255-x*40, 0)
			ImageSetPixel(src, x, y, c)
		}
	}
	data := writeTestTGA(t, src)
	back, err := ImageCreateFromTGA(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 4 || back.Height() != 3 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			sr, sg, sb, _ := ImageColorsForIndex(src, ImageColorAt(src, x, y))
			dr, dg, db, _ := ImageColorsForIndex(back, ImageColorAt(back, x, y))
			if sr != dr || sg != dg || sb != db {
				t.Errorf("(%d,%d) src=%d,%d,%d got=%d,%d,%d", x, y, sr, sg, sb, dr, dg, db)
			}
		}
	}
}

func TestTGARLERoundtrip(t *testing.T) {
	src := ImageCreateTrueColor(4, 4)
	ImageAlphaBlending(src, false)
	red := ImageColorAllocate(src, 200, 50, 25)
	ImageFilledRectangle(src, 0, 0, 3, 3, red)
	data := writeTestTGA_RLE(t, src)
	back, err := ImageCreateFromTGA(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 4 || back.Height() != 4 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	r, g, b, _ := ImageColorsForIndex(back, ImageColorAt(back, 1, 1))
	if r != 200 || g != 50 || b != 25 {
		t.Errorf("RLE pixel = %d,%d,%d", r, g, b)
	}
}

func TestTGABadHeader(t *testing.T) {
	// type 5 is undefined in TGA spec
	hdr := make([]byte, 18)
	hdr[2] = 5
	binary.LittleEndian.PutUint16(hdr[12:14], 2)
	binary.LittleEndian.PutUint16(hdr[14:16], 2)
	hdr[16] = 32
	if _, err := ImageCreateFromTGA(bytes.NewReader(hdr)); err == nil {
		t.Error("expected error for unsupported tga type")
	}
}
