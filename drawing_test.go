package gogd

import (
	"image"
	"testing"
)

func TestLineHorizontal(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 0, 0)
	ImageLine(img, 2, 5, 7, 5, c)
	for x := 2; x <= 7; x++ {
		if ImageColorAt(img, x, 5) != c {
			t.Errorf("missing pixel at (%d, 5)", x)
		}
	}
	if ImageColorAt(img, 1, 5) == c {
		t.Error("unexpected pixel at (1, 5)")
	}
}

func TestLineVertical(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 0, 255, 0)
	ImageLine(img, 3, 1, 3, 8, c)
	for y := 1; y <= 8; y++ {
		if ImageColorAt(img, 3, y) != c {
			t.Errorf("missing pixel at (3, %d)", y)
		}
	}
}

func TestLineDiagonal(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 0, 0, 255)
	ImageLine(img, 0, 0, 9, 9, c)
	for i := 0; i < 10; i++ {
		if ImageColorAt(img, i, i) != c {
			t.Errorf("missing pixel at (%d, %d)", i, i)
		}
	}
}

func TestDashedLine(t *testing.T) {
	img := ImageCreateTrueColor(20, 1)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 0, 0)
	black := ImageColorAllocate(img, 0, 0, 0)
	_ = black
	ImageDashedLine(img, 0, 0, 19, 0, c)
	drawn, skipped := 0, 0
	for x := 0; x < 20; x++ {
		if ImageColorAt(img, x, 0) == c {
			drawn++
		} else {
			skipped++
		}
	}
	if drawn == 0 || skipped == 0 {
		t.Errorf("dash pattern broken: drawn=%d skipped=%d", drawn, skipped)
	}
}

func TestRectangleOutline(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 255, 0)
	ImageRectangle(img, 2, 2, 7, 7, c)
	// Top edge
	for x := 2; x <= 7; x++ {
		if ImageColorAt(img, x, 2) != c {
			t.Errorf("top edge missing at (%d, 2)", x)
		}
	}
	// Interior should not be drawn
	if ImageColorAt(img, 4, 4) == c {
		t.Error("interior pixel should not be drawn")
	}
}

func TestFilledRectangle(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 0, 128, 255)
	ImageFilledRectangle(img, 2, 2, 5, 5, c)
	for y := 2; y <= 5; y++ {
		for x := 2; x <= 5; x++ {
			if ImageColorAt(img, x, y) != c {
				t.Errorf("missing fill at (%d, %d)", x, y)
			}
		}
	}
	if ImageColorAt(img, 1, 1) == c {
		t.Error("outside pixel filled")
	}
}

func TestFilledRectanglePalette(t *testing.T) {
	img := ImageCreate(10, 10)
	bg := ImageColorAllocate(img, 0, 0, 0)
	red := ImageColorAllocate(img, 255, 0, 0)
	_ = bg
	ImageFilledRectangle(img, 1, 1, 3, 3, red)
	if ImageColorAt(img, 2, 2) != red {
		t.Error("palette fill failed")
	}
	if ImageColorAt(img, 0, 0) != bg {
		t.Error("outside pixel changed")
	}
}

func TestPolygonTriangle(t *testing.T) {
	img := ImageCreateTrueColor(20, 20)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 0, 0)
	pts := []image.Point{{5, 2}, {15, 10}, {2, 18}}
	if !ImagePolygon(img, pts, c) {
		t.Fatal("ImagePolygon failed")
	}
	// Verify vertices are drawn
	if ImageColorAt(img, 5, 2) != c {
		t.Error("vertex (5,2) not drawn")
	}
}

func TestFilledPolygon(t *testing.T) {
	img := ImageCreateTrueColor(20, 20)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 0, 255, 0)
	pts := []image.Point{{2, 2}, {17, 2}, {17, 17}, {2, 17}}
	if !ImageFilledPolygon(img, pts, c) {
		t.Fatal("ImageFilledPolygon failed")
	}
	// Centre must be filled
	if ImageColorAt(img, 10, 10) != c {
		t.Error("centre not filled")
	}
}

func TestEllipseOutline(t *testing.T) {
	img := ImageCreateTrueColor(50, 50)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 0, 255)
	ImageEllipse(img, 25, 25, 40, 30, c)
	// Top of ellipse
	if ImageColorAt(img, 25, 25-15) != c {
		t.Error("top of ellipse missing")
	}
	// Centre should not be drawn (outline only)
	if ImageColorAt(img, 25, 25) == c {
		t.Error("centre should not be drawn")
	}
}

func TestFilledEllipse(t *testing.T) {
	img := ImageCreateTrueColor(50, 50)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 128, 128, 0)
	ImageFilledEllipse(img, 25, 25, 40, 30, c)
	if ImageColorAt(img, 25, 25) != c {
		t.Error("centre not filled")
	}
}

func TestArc(t *testing.T) {
	img := ImageCreateTrueColor(60, 60)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 128, 0)
	ImageArc(img, 30, 30, 40, 40, 0, 90, c)
	// Right-most point of the arc (0 degrees)
	if ImageColorAt(img, 50, 30) != c {
		t.Error("0-degree point missing")
	}
}

func TestFilledArcPie(t *testing.T) {
	img := ImageCreateTrueColor(60, 60)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 0, 128, 255)
	ImageFilledArc(img, 30, 30, 40, 40, 0, 90, c, ImgArcPie)
	// Centre should be filled (it's a pie slice)
	if ImageColorAt(img, 30, 30) != c {
		t.Error("centre of pie not filled")
	}
}

func TestFloodFill(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	blue := ImageColorAllocate(img, 0, 0, 255)
	// Draw a red border rectangle
	ImageRectangle(img, 2, 2, 7, 7, red)
	// Fill inside the rectangle with blue
	ImageFill(img, 4, 4, blue)
	if ImageColorAt(img, 4, 4) != blue {
		t.Error("fill target not reached")
	}
	// Border should remain red
	if ImageColorAt(img, 2, 2) != red {
		t.Error("border pixel changed")
	}
}

func TestFillToBorder(t *testing.T) {
	img := ImageCreateTrueColor(10, 10)
	ImageAlphaBlending(img, false)
	red := ImageColorAllocate(img, 255, 0, 0)
	green := ImageColorAllocate(img, 0, 255, 0)
	ImageRectangle(img, 2, 2, 7, 7, red)
	ImageFillToBorder(img, 4, 4, red, green)
	if ImageColorAt(img, 4, 4) != green {
		t.Error("fill target not reached")
	}
	// Outside rectangle should not be filled
	if ImageColorAt(img, 0, 0) == green {
		t.Error("fill escaped the border")
	}
}

func TestThickness(t *testing.T) {
	img := ImageCreateTrueColor(20, 20)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 255, 255)
	ImageSetThickness(img, 3)
	ImageLine(img, 10, 10, 10, 10, c)
	// With thickness 3, pixels at offsets -1 and +1 should also be drawn
	if ImageColorAt(img, 9, 10) != c {
		t.Error("thick pixel at (-1, 0) missing")
	}
	if ImageColorAt(img, 10, 9) != c {
		t.Error("thick pixel at (0, -1) missing")
	}
}

func TestClip(t *testing.T) {
	img := ImageCreateTrueColor(20, 20)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 0, 0)
	ImageSetClip(img, 5, 5, 14, 14)
	ImageLine(img, 0, 10, 19, 10, c)
	// Pixel outside clip should not be drawn
	if ImageColorAt(img, 3, 10) == c {
		t.Error("pixel outside clip drawn")
	}
	// Pixel inside clip should be drawn
	if ImageColorAt(img, 10, 10) != c {
		t.Error("pixel inside clip not drawn")
	}
	// Verify getclip
	x1, y1, x2, y2 := ImageGetClip(img)
	if x1 != 5 || y1 != 5 || x2 != 14 || y2 != 14 {
		t.Errorf("clip = (%d,%d,%d,%d)", x1, y1, x2, y2)
	}
}

func TestSetStyleLineAlternating(t *testing.T) {
	img := ImageCreateTrueColor(12, 1)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	red := ImageColorAllocate(img, 255, 0, 0)
	_ = black
	// Alternate red / transparent so every other pixel stays untouched.
	ImageSetStyle(img, []Color{red, ColorTransparent})
	ImageLine(img, 0, 0, 11, 0, ColorStyled)
	for x := 0; x < 12; x++ {
		r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, x, 0))
		if x%2 == 0 && r != 255 {
			t.Errorf("expected red at x=%d, got r=%d", x, r)
		}
		if x%2 == 1 && r == 255 {
			t.Errorf("expected untouched at x=%d, got r=%d", x, r)
		}
	}
}

func TestSetBrushStamped(t *testing.T) {
	brush := ImageCreateTrueColor(3, 3)
	ImageAlphaBlending(brush, false)
	red := ImageColorAllocate(brush, 255, 0, 0)
	ImageFilledRectangle(brush, 0, 0, 2, 2, red)

	img := ImageCreateTrueColor(12, 12)
	ImageAlphaBlending(img, false)
	black := ImageColorAllocate(img, 0, 0, 0)
	ImageFilledRectangle(img, 0, 0, 11, 11, black)
	ImageSetBrush(img, brush)
	ImageLine(img, 4, 4, 7, 4, ColorBrushed)
	// The brush (3×3 red) should have covered pixels around y=4 centred at x in [4,7].
	if r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 5, 4)); r != 255 {
		t.Errorf("brush stamp missing at (5,4): r=%d", r)
	}
	if r, _, _, _ := ImageColorsForIndex(img, ImageColorAt(img, 5, 3)); r != 255 {
		t.Errorf("brush stamp row missing at (5,3): r=%d", r)
	}
}

func TestOpenPolygon(t *testing.T) {
	img := ImageCreateTrueColor(20, 20)
	ImageAlphaBlending(img, false)
	c := ImageColorAllocate(img, 255, 0, 0)
	pts := []image.Point{{0, 0}, {10, 0}, {10, 10}}
	ImageOpenPolygon(img, pts, c)
	// Closing segment (10,10)→(0,0) should NOT be drawn
	if ImageColorAt(img, 5, 5) == c {
		t.Error("diagonal closing segment drawn for open polygon")
	}
}
