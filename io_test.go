package gogd

import (
	"bytes"
	"testing"
)

func fillTruecolor(img *Image, r, g, b int) Color {
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, r, g, b)
	for y := 0; y < img.Height(); y++ {
		for x := 0; x < img.Width(); x++ {
			ImageSetPixel(img, x, y, c)
		}
	}
	return c
}

func TestPNGRoundtripTrueColor(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	fillTruecolor(img, 255, 128, 64)

	var buf bytes.Buffer
	if err := ImagePNG(img, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromPNG(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 10 || back.Height() != 10 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	r, g, b, a := ImageColorsForIndex(back, ImageColorAt(back, 3, 3))
	if r != 255 || g != 128 || b != 64 || a != AlphaOpaque {
		t.Errorf("pixel = %d,%d,%d,%d", r, g, b, a)
	}
}

func TestJPEGEncodeDecode(t *testing.T) {
	img := ImageCreateTrueColor(8, 8)
	fillTruecolor(img, 255, 255, 255)

	var buf bytes.Buffer
	if err := ImageJPEG(img, &buf, 90); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromJPEG(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !back.IsTrueColor() {
		t.Error("JPEG should decode to truecolor")
	}
	if back.Width() != 8 || back.Height() != 8 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
}

func TestJPEGQualityClamping(t *testing.T) {
	img := ImageCreateTrueColor(2, 2)
	var buf bytes.Buffer
	// Values outside 0..100 should not produce encoder errors.
	if err := ImageJPEG(img, &buf, 200); err != nil {
		t.Fatalf("over-range quality rejected: %v", err)
	}
	if err := ImageJPEG(img, &bytes.Buffer{}, -1); err != nil {
		t.Fatalf("default quality rejected: %v", err)
	}
}

func TestGIFPaletteRoundtrip(t *testing.T) {
	img := ImageCreate(4, 4)
	_ = ImageColorAllocate(img, 0, 0, 0)
	red := ImageColorAllocate(img, 255, 0, 0)
	ImageSetPixel(img, 2, 2, red)

	var buf bytes.Buffer
	if err := ImageGIF(img, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromGIF(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.IsTrueColor() {
		t.Error("GIF should decode to a palette image")
	}
	r, g, b, _ := ImageColorsForIndex(back, ImageColorAt(back, 2, 2))
	if r != 255 || g != 0 || b != 0 {
		t.Errorf("red pixel = %d,%d,%d", r, g, b)
	}
}

func TestBMPRoundtrip(t *testing.T) {
	img := ImageCreateTrueColor(3, 3)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 10, 20, 30)
	ImageSetPixel(img, 1, 1, c)

	var buf bytes.Buffer
	if err := ImageBMP(img, &buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := ImageCreateFromBMP(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	r, g, b, _ := ImageColorsForIndex(back, ImageColorAt(back, 1, 1))
	if r != 10 || g != 20 || b != 30 {
		t.Errorf("bmp pixel = %d,%d,%d", r, g, b)
	}
}

func TestImageCreateFromString(t *testing.T) {
	img := ImageCreateTrueColor(5, 5)
	fillTruecolor(img, 12, 34, 56)

	var buf bytes.Buffer
	if err := ImagePNG(img, &buf); err != nil {
		t.Fatal(err)
	}
	back, err := ImageCreateFromString(buf.Bytes())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.Width() != 5 || back.Height() != 5 {
		t.Errorf("size = %dx%d", back.Width(), back.Height())
	}
	r, g, b, _ := ImageColorsForIndex(back, ImageColorAt(back, 0, 0))
	if r != 12 || g != 34 || b != 56 {
		t.Errorf("pixel = %d,%d,%d", r, g, b)
	}
}

func TestImageCreateFromStringBadData(t *testing.T) {
	if _, err := ImageCreateFromString([]byte("not an image")); err == nil {
		t.Error("expected error for garbage data")
	}
}

func TestGetImageSizeFromString(t *testing.T) {
	for _, tc := range []struct {
		name   string
		encode func(img *Image, w *bytes.Buffer) error
		typ    int
		mime   string
	}{
		{"png", func(img *Image, w *bytes.Buffer) error { return ImagePNG(img, w) }, ImageTypePNG, "image/png"},
		{"jpeg", func(img *Image, w *bytes.Buffer) error { return ImageJPEG(img, w, 75) }, ImageTypeJPEG, "image/jpeg"},
		{"gif", func(img *Image, w *bytes.Buffer) error { return ImageGIF(img, w) }, ImageTypeGIF, "image/gif"},
		{"bmp", func(img *Image, w *bytes.Buffer) error { return ImageBMP(img, w) }, ImageTypeBMP, "image/bmp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := ImageCreateTrueColor(17, 23)
			var buf bytes.Buffer
			if err := tc.encode(img, &buf); err != nil {
				t.Fatalf("encode: %v", err)
			}
			info, err := GetImageSizeFromString(buf.Bytes())
			if err != nil {
				t.Fatalf("info: %v", err)
			}
			if info.Width != 17 || info.Height != 23 {
				t.Errorf("size = %dx%d", info.Width, info.Height)
			}
			if info.Type != tc.typ {
				t.Errorf("type = %d, want %d", info.Type, tc.typ)
			}
			if info.MimeType != tc.mime {
				t.Errorf("mime = %q, want %q", info.MimeType, tc.mime)
			}
		})
	}
}

func TestImageTypeToExtension(t *testing.T) {
	cases := []struct {
		typ      int
		withDot  string
		noDot    string
	}{
		{ImageTypePNG, ".png", "png"},
		{ImageTypeJPEG, ".jpeg", "jpeg"},
		{ImageTypeGIF, ".gif", "gif"},
		{ImageTypeBMP, ".bmp", "bmp"},
		{ImageTypeWEBP, ".webp", "webp"},
		{999, "", ""},
	}
	for _, c := range cases {
		if got := ImageTypeToExtension(c.typ, true); got != c.withDot {
			t.Errorf("typ=%d withDot: got %q want %q", c.typ, got, c.withDot)
		}
		if got := ImageTypeToExtension(c.typ, false); got != c.noDot {
			t.Errorf("typ=%d noDot: got %q want %q", c.typ, got, c.noDot)
		}
	}
}

func TestImageTypeToMimeType(t *testing.T) {
	if m := ImageTypeToMimeType(ImageTypePNG); m != "image/png" {
		t.Errorf("png mime = %q", m)
	}
	if m := ImageTypeToMimeType(ImageTypeWEBP); m != "image/webp" {
		t.Errorf("webp mime = %q", m)
	}
	if m := ImageTypeToMimeType(999); m != "application/octet-stream" {
		t.Errorf("unknown mime = %q", m)
	}
}

func TestImageTypesReportsSupported(t *testing.T) {
	bits := ImageTypes()
	for _, want := range []int{ImgPNG, ImgJPEG, ImgGIF, ImgBMP, ImgWEBP} {
		if bits&want == 0 {
			t.Errorf("missing format bit %d", want)
		}
	}
}

func TestEncodeNilImage(t *testing.T) {
	var buf bytes.Buffer
	if err := ImagePNG(nil, &buf); err == nil {
		t.Error("expected error on nil image")
	}
}
