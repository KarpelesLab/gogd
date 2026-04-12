package gogd

import (
	"bytes"
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

func TestGDBadMagic(t *testing.T) {
	if _, err := ImageCreateFromGD(bytes.NewReader([]byte{0x00, 0x00})); err == nil {
		t.Error("expected error for bad magic")
	}
}
