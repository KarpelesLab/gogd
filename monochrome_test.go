package gogd

import (
	"bytes"
	"strings"
	"testing"
)

func makeCheckerboard(w, h int) *Image {
	img := ImageCreateTrueColor(w, h)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	white := ImageColorAllocate(img, 255, 255, 255)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := black
			if (x+y)%2 == 0 {
				c = white
			}
			ImageSetPixel(img, x, y, c)
		}
	}
	return img
}

func TestWBMPRoundtrip(t *testing.T) {
	img := makeCheckerboard(16, 10)
	var buf bytes.Buffer
	if err := ImageWBMP(img, &buf, ColorNone); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if buf.Len() < 4 {
		t.Fatalf("wbmp too short: %d bytes", buf.Len())
	}
	back, err := ImageCreateFromWBMP(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 16 || back.Height() != 10 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	// The white (0,0) pixel should round-trip to white.
	r, _, _, _ := ImageColorsForIndex(back, ImageColorAt(back, 0, 0))
	if r != 255 {
		t.Errorf("(0,0) = %d, want 255 (white)", r)
	}
	r, _, _, _ = ImageColorsForIndex(back, ImageColorAt(back, 1, 0))
	if r != 0 {
		t.Errorf("(1,0) = %d, want 0 (black)", r)
	}
}

func TestWBMPLargeDimensions(t *testing.T) {
	// Exercise the multi-byte varint path with a >127-wide image.
	img := makeCheckerboard(200, 180)
	var buf bytes.Buffer
	if err := ImageWBMP(img, &buf, ColorNone); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromWBMP(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 200 || back.Height() != 180 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
}

func TestWBMPVarint(t *testing.T) {
	for _, v := range []int{0, 1, 127, 128, 200, 16383, 16384, 2097151} {
		var buf bytes.Buffer
		bw := &singleByteWriter{w: &buf}
		if err := writeWBMPInt(bw, v); err != nil {
			t.Fatalf("write %d: %v", v, err)
		}
		br := &singleByteReader{r: &buf}
		got, err := readWBMPInt(br)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if got != v {
			t.Errorf("roundtrip %d -> %d", v, got)
		}
	}
}

// Minimal io.ByteWriter / io.ByteReader over a bytes.Buffer.
type singleByteWriter struct{ w *bytes.Buffer }

func (s *singleByteWriter) WriteByte(b byte) error { s.w.WriteByte(b); return nil }

type singleByteReader struct{ r *bytes.Buffer }

func (s *singleByteReader) ReadByte() (byte, error) { return s.r.ReadByte() }

func TestXBMRoundtrip(t *testing.T) {
	img := makeCheckerboard(13, 9)
	var buf bytes.Buffer
	if err := ImageXBM(img, &buf, ColorNone, "test"); err != nil {
		t.Fatalf("encode: %v", err)
	}
	text := buf.String()
	if !strings.Contains(text, "#define test_width 13") {
		t.Errorf("missing width define; got:\n%s", text)
	}
	if !strings.Contains(text, "#define test_height 9") {
		t.Errorf("missing height define")
	}
	if !strings.Contains(text, "test_bits[]") {
		t.Errorf("missing bits array")
	}

	back, err := ImageCreateFromXBM(strings.NewReader(text))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 13 || back.Height() != 9 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	// (0,0) of checkerboard is white → XBM decodes to white.
	r, _, _, _ := ImageColorsForIndex(back, ImageColorAt(back, 0, 0))
	if r != 255 {
		t.Errorf("(0,0) = %d, want 255", r)
	}
}

func TestXBMDecodeCanonical(t *testing.T) {
	// Canonical 8x2 XBM with a diagonal-ish pattern.
	src := `#define foo_width 8
#define foo_height 2
static unsigned char foo_bits[] = {
  0x01, 0x80
};
`
	img, err := ImageCreateFromXBM(strings.NewReader(src))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Width() != 8 || img.Height() != 2 {
		t.Errorf("size = %dx%d", img.Width(), img.Height())
	}
	// Byte 0 is 0x01, LSB-first → bit 0 is set → (0, 0) is foreground (black).
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 0 {
		t.Errorf("(0, 0) = %d, want 0", r)
	}
	// (1, 0) should be background (white).
	r, _, _, _ = ImageColorsForIndex(img, ImageColorAt(img, 1, 0))
	if r != 255 {
		t.Errorf("(1, 0) = %d, want 255", r)
	}
	// Byte 1 is 0x80 → bit 7 set → (7, 1) is foreground.
	r, _, _, _ = ImageColorsForIndex(img, ImageColorAt(img, 7, 1))
	if r != 0 {
		t.Errorf("(7, 1) = %d, want 0", r)
	}
}

func TestXBMMalformed(t *testing.T) {
	if _, err := ImageCreateFromXBM(strings.NewReader("not an xbm")); err == nil {
		t.Error("expected error for malformed xbm")
	}
}
