package gogd

import (
	"strings"
	"testing"
)

const xpm3x3 = `/* XPM */
static char *test[] = {
"3 3 3 1",
"  c none",
"X c #ff0000",
"o c #00ff00",
"XoX",
"oXo",
"XoX"
};`

func TestXPMDecode(t *testing.T) {
	img, err := ImageCreateFromXPM(strings.NewReader(xpm3x3))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Width() != 3 || img.Height() != 3 {
		t.Errorf("size = %dx%d", img.Width(), img.Height())
	}
	// (0, 0) was 'X' = #ff0000 = red.
	r, g, b, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 255 || g != 0 || b != 0 {
		t.Errorf("(0,0) = %d,%d,%d", r, g, b)
	}
	// (1, 0) was 'o' = #00ff00 = green.
	r, g, b, _ = ImageColorsForIndex(img, ImageColorAt(img, 1, 0))
	if r != 0 || g != 255 || b != 0 {
		t.Errorf("(1,0) = %d,%d,%d", r, g, b)
	}
}

func TestXPMShortHex(t *testing.T) {
	src := `/* XPM */
static char *t[] = {
"1 1 1 1",
"x c #f00",
"x"
};`
	img, err := ImageCreateFromXPM(strings.NewReader(src))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r < 250 {
		t.Errorf("short-form red = %d", r)
	}
}

func TestXPMNamedColor(t *testing.T) {
	src := `/* XPM */
static char *t[] = {
"1 1 1 1",
"x c red",
"x"
};`
	img, err := ImageCreateFromXPM(strings.NewReader(src))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 0, 0))
	if r != 255 {
		t.Errorf("named red = %d", r)
	}
}

func TestXPMMalformed(t *testing.T) {
	if _, err := ImageCreateFromXPM(strings.NewReader("not xpm")); err == nil {
		t.Error("expected error")
	}
}
