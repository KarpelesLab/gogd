package gogd

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"
)

func TestGDTrueColorRoundtrip(t *testing.T) {
	src := ImageCreateTrueColor(5, 3)
	ImageAlphaBlending(src, false)
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			c := ImageColorAllocateAlpha(src, x*50, y*80, 255-x*40, x%4*30)
			ImageSetPixel(src, x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := ImageGD(src, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromGD(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 5 || back.Height() != 3 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	if !back.IsTrueColor() {
		t.Error("decoded truecolor as non-truecolor")
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			sr, sg, sb, sa := ImageColorsForIndex(src, ImageColorAt(src, x, y))
			dr, dg, db, da := ImageColorsForIndex(back, ImageColorAt(back, x, y))
			if sr != dr || sg != dg || sb != db {
				t.Errorf("(%d,%d) RGB src=%d,%d,%d back=%d,%d,%d", x, y, sr, sg, sb, dr, dg, db)
			}
			// Alpha may drift one step through the 0..127 → 0..255 → 0..127
			// quantisation. Accept ±1.
			if da > sa+1 || da < sa-1 {
				t.Errorf("(%d,%d) alpha src=%d back=%d", x, y, sa, da)
			}
		}
	}
}

func TestGDPaletteRoundtrip(t *testing.T) {
	src := ImageCreate(4, 2)
	black := ImageColorAllocate(src, 0, 0, 0)
	red := ImageColorAllocate(src, 255, 0, 0)
	ImageFilledRectangle(src, 0, 0, 1, 1, black)
	ImageFilledRectangle(src, 2, 0, 3, 1, red)

	var buf bytes.Buffer
	if err := ImageGD(src, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromGD(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.IsTrueColor() {
		t.Error("palette decoded as truecolor")
	}
	if ImageColorsTotal(back) != 2 {
		t.Errorf("palette size = %d", ImageColorsTotal(back))
	}
	// (2, 0) was red
	r, _, _, _ := ImageColorsForIndex(back, ImageColorAt(back, 2, 0))
	if r != 255 {
		t.Errorf("red pixel = %d", r)
	}
}

// writeRawGD2 synthesises a minimal format-3 (raw truecolor) GD2 file
// so the reader has something valid to chew on without pulling in a
// compressed writer.
func writeRawGD2(src *Image) []byte {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var buf bytes.Buffer
	buf.Write([]byte("gd2\x00"))
	binary.Write(&buf, binary.BigEndian, uint16(2)) // version
	binary.Write(&buf, binary.BigEndian, uint16(w))
	binary.Write(&buf, binary.BigEndian, uint16(h))
	binary.Write(&buf, binary.BigEndian, uint16(64)) // chunk size (ignored for raw)
	binary.Write(&buf, binary.BigEndian, uint16(3))  // format: raw truecolor
	binary.Write(&buf, binary.BigEndian, uint16(1))  // nchunks x (irrelevant for raw)
	binary.Write(&buf, binary.BigEndian, uint16(1))  // nchunks y
	buf.WriteByte(1)                                 // truecolor flag
	binary.Write(&buf, binary.BigEndian, int32(-1))  // transparent
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.nrgba.NRGBAAt(x, y)
			buf.WriteByte(byte(stdAlphaToGD(c.A)))
			buf.WriteByte(c.R)
			buf.WriteByte(c.G)
			buf.WriteByte(c.B)
		}
	}
	return buf.Bytes()
}

func TestGD2TrueColorRaw(t *testing.T) {
	src := ImageCreateTrueColor(3, 2)
	ImageAlphaBlending(src, false)
	ImageSetPixel(src, 0, 0, ImageColorAllocate(src, 255, 0, 0))
	ImageSetPixel(src, 1, 0, ImageColorAllocate(src, 0, 255, 0))
	ImageSetPixel(src, 2, 0, ImageColorAllocate(src, 0, 0, 255))
	ImageSetPixel(src, 0, 1, ImageColorAllocateAlpha(src, 10, 20, 30, 32))
	data := writeRawGD2(src)
	back, err := ImageCreateFromGD2(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 3 || back.Height() != 2 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	r, g, b, _ := ImageColorsForIndex(back, ImageColorAt(back, 1, 0))
	if r != 0 || g != 255 || b != 0 {
		t.Errorf("(1,0) = %d,%d,%d", r, g, b)
	}
}

func TestGD2TrueColorCompressed(t *testing.T) {
	// 4×4 truecolor image, chunk size 2 → 2×2 grid.
	w, h, cs := 4, 4, 2
	bpp := 4
	// Build the raw pixel buffer first so we can copy chunk regions from it.
	type nrgba struct{ a, r, g, b byte }
	pix := make([]nrgba, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			pix[y*w+x] = nrgba{0, byte(x * 60), byte(y * 60), byte((x + y) * 30)}
		}
	}

	// Build header + truecolor info.
	var out bytes.Buffer
	out.Write([]byte("gd2\x00"))
	binary.Write(&out, binary.BigEndian, uint16(2))
	binary.Write(&out, binary.BigEndian, uint16(w))
	binary.Write(&out, binary.BigEndian, uint16(h))
	binary.Write(&out, binary.BigEndian, uint16(cs))
	binary.Write(&out, binary.BigEndian, uint16(4)) // compressed truecolor
	nCx := (w + cs - 1) / cs
	nCy := (h + cs - 1) / cs
	binary.Write(&out, binary.BigEndian, uint16(nCx))
	binary.Write(&out, binary.BigEndian, uint16(nCy))
	out.WriteByte(1)                                // truecolor
	binary.Write(&out, binary.BigEndian, int32(-1)) // transparent

	// Emit chunk index table (placeholder — we'll fill in real offsets below).
	tableOffset := out.Len()
	tableSize := nCx * nCy * 8
	out.Write(make([]byte, tableSize))

	// Emit compressed chunks and record their offset/size.
	offsets := make([]uint32, nCx*nCy)
	sizes := make([]uint32, nCx*nCy)
	for cy := 0; cy < nCy; cy++ {
		for cx := 0; cx < nCx; cx++ {
			chunk := new(bytes.Buffer)
			for row := 0; row < cs; row++ {
				for col := 0; col < cs; col++ {
					sx := cx*cs + col
					sy := cy*cs + row
					if sx >= w || sy >= h {
						// Pad with zeroes so chunk is always cs*cs.
						chunk.Write(make([]byte, bpp))
						continue
					}
					p := pix[sy*w+sx]
					chunk.WriteByte(p.a)
					chunk.WriteByte(p.r)
					chunk.WriteByte(p.g)
					chunk.WriteByte(p.b)
				}
			}
			compressed := new(bytes.Buffer)
			zw := zlib.NewWriter(compressed)
			zw.Write(chunk.Bytes())
			zw.Close()

			offsets[cy*nCx+cx] = uint32(out.Len())
			sizes[cy*nCx+cx] = uint32(compressed.Len())
			out.Write(compressed.Bytes())
		}
	}

	// Rewrite the chunk index table.
	raw := out.Bytes()
	for i := range offsets {
		binary.BigEndian.PutUint32(raw[tableOffset+i*8:], offsets[i])
		binary.BigEndian.PutUint32(raw[tableOffset+i*8+4:], sizes[i])
	}

	back, err := ImageCreateFromGD2(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != w || back.Height() != h {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	// Spot-check a few pixels.
	want := pix[1*w+2]
	r, g, b, _ := ImageColorsForIndex(back, ImageColorAt(back, 2, 1))
	if byte(r) != want.r || byte(g) != want.g || byte(b) != want.b {
		t.Errorf("(2,1) = %d,%d,%d, want %d,%d,%d", r, g, b, want.r, want.g, want.b)
	}
}

func TestGD2BadMagic(t *testing.T) {
	bad := bytes.Repeat([]byte{0}, 32)
	if _, err := ImageCreateFromGD2(bytes.NewReader(bad)); err == nil {
		t.Error("expected error for bad magic")
	}
}

func TestGDBadMagic(t *testing.T) {
	if _, err := ImageCreateFromGD(bytes.NewReader([]byte{0x00, 0x00})); err == nil {
		t.Error("expected error for bad magic")
	}
}
