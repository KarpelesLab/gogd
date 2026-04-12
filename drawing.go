package gogd

import (
	"image"
	"image/draw"
	"math"
)

// Special color sentinels matching PHP's IMG_COLOR_* constants.
// Pass them to drawing functions in place of a real color to switch
// behaviour: styled (step through the pattern set via [ImageSetStyle]),
// brushed (paint a small image at each step), tiled (fill with a tile),
// or transparent (skip the pixel in a styled pattern).
const (
	ColorStyled        Color = -2
	ColorBrushed       Color = -3
	ColorStyledBrushed Color = -4
	ColorTiled         Color = -5
	ColorTransparent   Color = -6
)

// --- state ---

// ImageSetStyle sets the color pattern used when a drawing function is
// invoked with [ColorStyled]. Each entry is consulted in turn per pixel
// step; [ColorTransparent] entries leave the underlying pixel alone.
func ImageSetStyle(img *Image, style []Color) bool {
	if img == nil {
		return false
	}
	img.style = append(img.style[:0], style...)
	return true
}

// ImageSetBrush records a brush image to paint at each pixel when a
// drawing function is invoked with [ColorBrushed]. Currently stored for
// API compatibility; rendering is pending.
func ImageSetBrush(img *Image, brush *Image) bool {
	if img == nil {
		return false
	}
	img.brush = brush
	return true
}

// ImageSetTile records a tile image used for area fills when a drawing
// function is invoked with [ColorTiled]. Currently stored for API
// compatibility; rendering is pending.
func ImageSetTile(img *Image, tile *Image) bool {
	if img == nil {
		return false
	}
	img.tile = tile
	return true
}

// ImageSetThickness sets the line thickness in pixels used for outline
// operations (lines, rectangles, ellipses, polygons). Returns the
// previous value.
func ImageSetThickness(img *Image, thickness int) int {
	if img == nil {
		return 0
	}
	if thickness < 1 {
		thickness = 1
	}
	prev := img.thickness
	img.thickness = thickness
	return prev
}

// ImageAntialias toggles the antialias flag. This is currently a no-op
// for actual rendering, kept for API completeness.
func ImageAntialias(img *Image, enable bool) bool {
	if img == nil {
		return false
	}
	prev := img.antialias
	img.antialias = enable
	return prev
}

// ImageSetClip sets the drawing clip rectangle. Coordinates are inclusive
// on both ends. All drawing operations are restricted to pixels inside it.
func ImageSetClip(img *Image, x1, y1, x2, y2 int) bool {
	if img == nil {
		return false
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	img.clip = image.Rect(x1, y1, x2+1, y2+1)
	return true
}

// ImageGetClip returns the effective clip rectangle as (x1, y1, x2, y2)
// with inclusive coordinates.
func ImageGetClip(img *Image) (x1, y1, x2, y2 int) {
	if img == nil {
		return 0, 0, 0, 0
	}
	r := img.clipRect()
	return r.Min.X, r.Min.Y, r.Max.X - 1, r.Max.Y - 1
}

// clipRect returns the effective clip rectangle (clip ∩ bounds),
// defaulting to the full image bounds when no clip has been set.
func (img *Image) clipRect() image.Rectangle {
	b := img.Bounds()
	if img.clip.Empty() {
		return b
	}
	return img.clip.Intersect(b)
}

// --- internal plot helpers ---

func (img *Image) plotPixel(x, y int, c Color) {
	if !(image.Point{X: x, Y: y}).In(img.clipRect()) {
		return
	}
	ImageSetPixel(img, x, y, c)
}

func (img *Image) plotThick(x, y int, c Color) {
	th := img.thickness
	if th <= 1 {
		img.plotPixel(x, y, c)
		return
	}
	r := th / 2
	for dy := -r; dy < th-r; dy++ {
		for dx := -r; dx < th-r; dx++ {
			img.plotPixel(x+dx, y+dy, c)
		}
	}
}

// writeColor sets a pixel without alpha blending. Used by flood fill.
func (img *Image) writeColor(x, y int, c Color) {
	if img.nrgba != nil {
		img.nrgba.SetNRGBA(x, y, gdColorToNRGBA(c))
		return
	}
	if img.pal != nil && int(c) >= 0 && int(c) < len(img.pal.Palette) {
		img.pal.SetColorIndex(x, y, uint8(c))
		return
	}
	if img.generic != nil {
		img.generic.Set(x, y, gdColorToNRGBA(c))
	}
}

// --- lines ---

// ImageLine draws a line from (x1, y1) to (x2, y2) using Bresenham's
// algorithm, respecting the current thickness and clip rectangle.
// Accepts any [draw.Image]; state (thickness, clip) is taken from
// gogd's *Image or defaults to 1 / whole image otherwise.
func ImageLine(dst draw.Image, x1, y1, x2, y2 int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	dx, dy := iabs(x2-x1), -iabs(y2-y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx + dy
	step := 0
	for {
		img.plotStyled(x1, y1, c, step)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			if x1 == x2 {
				break
			}
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			if y1 == y2 {
				break
			}
			err += dx
			y1 += sy
		}
		step++
	}
	return true
}

// plotStyled dispatches on special color sentinels. For plain Color
// values it calls plotThick directly; for ColorStyled it cycles through
// img.style; for ColorBrushed/ColorStyledBrushed it stamps the brush;
// ColorTransparent skips the pixel.
func (img *Image) plotStyled(x, y int, c Color, step int) {
	switch c {
	case ColorTransparent:
		return
	case ColorStyled:
		if len(img.style) == 0 {
			return
		}
		sc := img.style[step%len(img.style)]
		if sc == ColorTransparent {
			return
		}
		img.plotThick(x, y, sc)
	case ColorBrushed:
		img.stampBrush(x, y)
	case ColorStyledBrushed:
		if len(img.style) == 0 {
			return
		}
		if img.style[step%len(img.style)] == ColorTransparent {
			return
		}
		img.stampBrush(x, y)
	default:
		img.plotThick(x, y, c)
	}
}

// tileFillRect fills the given rectangle with a repeating copy of
// img.tile. Returns false if no tile has been set.
func tileFillRect(img *Image, r image.Rectangle) bool {
	if img.tile == nil {
		return false
	}
	tb := img.tile.Bounds()
	tw, th := tb.Dx(), tb.Dy()
	if tw <= 0 || th <= 0 {
		return false
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			tx := tb.Min.X + ((x%tw)+tw)%tw
			ty := tb.Min.Y + ((y%th)+th)%th
			img.Set(x, y, img.tile.At(tx, ty))
		}
	}
	return true
}

// stampBrush paints the brush image centred at (cx, cy).
func (img *Image) stampBrush(cx, cy int) {
	if img.brush == nil {
		return
	}
	bb := img.brush.Bounds()
	w, h := bb.Dx(), bb.Dy()
	ox := cx - w/2
	oy := cy - h/2
	for by := 0; by < h; by++ {
		for bx := 0; bx < w; bx++ {
			px, py := ox+bx, oy+by
			if !(image.Point{X: px, Y: py}).In(img.clipRect()) {
				continue
			}
			c := img.brush.At(bb.Min.X+bx, bb.Min.Y+by)
			img.Set(px, py, c)
		}
	}
}

// ImageDashedLine draws a dashed line (4 on, 4 off) from (x1, y1) to
// (x2, y2). Accepts any [draw.Image].
func ImageDashedLine(dst draw.Image, x1, y1, x2, y2 int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	dx, dy := iabs(x2-x1), -iabs(y2-y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx + dy
	step := 0
	for {
		if step%8 < 4 {
			img.plotThick(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			if x1 == x2 {
				break
			}
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			if y1 == y2 {
				break
			}
			err += dx
			y1 += sy
		}
		step++
	}
	return true
}

// --- rectangles ---

// ImageRectangle draws an outlined rectangle. Accepts any [draw.Image].
func ImageRectangle(dst draw.Image, x1, y1, x2, y2 int, c Color) bool {
	if dst == nil {
		return false
	}
	ImageLine(dst, x1, y1, x2, y1, c)
	ImageLine(dst, x2, y1, x2, y2, c)
	ImageLine(dst, x2, y2, x1, y2, c)
	ImageLine(dst, x1, y2, x1, y1, c)
	return true
}

// ImageFilledRectangle draws a filled rectangle from (x1, y1) to
// (x2, y2) inclusive. For truecolor images, alpha blending is honoured.
// Accepts any [draw.Image]. Pass [ColorTiled] to fill with the tile set
// via [ImageSetTile].
func ImageFilledRectangle(dst draw.Image, x1, y1, x2, y2 int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	r := image.Rect(x1, y1, x2+1, y2+1).Intersect(img.clipRect())
	if r.Empty() {
		return true
	}
	if c == ColorTiled {
		return tileFillRect(img, r)
	}
	if img.nrgba != nil {
		src := image.NewUniform(gdColorToNRGBA(c))
		op := draw.Src
		if img.alphaBlending {
			op = draw.Over
		}
		draw.Draw(img.nrgba, r, src, image.Point{}, op)
		return true
	}
	if img.pal != nil && int(c) >= 0 && int(c) < len(img.pal.Palette) {
		idx := uint8(c)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.pal.SetColorIndex(x, y, idx)
			}
		}
		return true
	}
	if img.generic != nil {
		src := image.NewUniform(gdColorToNRGBA(c))
		op := draw.Src
		if img.alphaBlending {
			op = draw.Over
		}
		draw.Draw(img.generic, r, src, image.Point{}, op)
		return true
	}
	return false
}

// --- polygons ---

// ImagePolygon draws a closed polygon.
func ImagePolygon(dst draw.Image, points []image.Point, c Color) bool {
	return drawPolygonOutline(dst, points, c, true)
}

// ImageOpenPolygon draws an open polygon (no line from last to first).
func ImageOpenPolygon(dst draw.Image, points []image.Point, c Color) bool {
	return drawPolygonOutline(dst, points, c, false)
}

func drawPolygonOutline(dst draw.Image, points []image.Point, c Color, closed bool) bool {
	if dst == nil || len(points) < 2 {
		return false
	}
	for i := 0; i < len(points)-1; i++ {
		ImageLine(dst, points[i].X, points[i].Y, points[i+1].X, points[i+1].Y, c)
	}
	if closed && len(points) >= 3 {
		last := len(points) - 1
		ImageLine(dst, points[last].X, points[last].Y, points[0].X, points[0].Y, c)
	}
	return true
}

// ImageFilledPolygon draws a filled polygon using scanline fill.
func ImageFilledPolygon(dst draw.Image, points []image.Point, c Color) bool {
	img := asImage(dst)
	if img == nil || len(points) < 3 {
		return false
	}
	miny, maxy := points[0].Y, points[0].Y
	for _, p := range points[1:] {
		if p.Y < miny {
			miny = p.Y
		}
		if p.Y > maxy {
			maxy = p.Y
		}
	}
	clip := img.clipRect()
	if miny < clip.Min.Y {
		miny = clip.Min.Y
	}
	if maxy >= clip.Max.Y {
		maxy = clip.Max.Y - 1
	}

	n := len(points)
	for y := miny; y <= maxy; y++ {
		var xs []int
		j := n - 1
		for i := 0; i < n; i++ {
			pi, pj := points[i], points[j]
			if (pi.Y <= y && pj.Y > y) || (pj.Y <= y && pi.Y > y) {
				x := pi.X + (y-pi.Y)*(pj.X-pi.X)/(pj.Y-pi.Y)
				xs = append(xs, x)
			}
			j = i
		}
		sortInts(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			for x := xs[k]; x <= xs[k+1]; x++ {
				img.plotPixel(x, y, c)
			}
		}
	}
	return true
}

// --- ellipse ---

// ImageEllipse draws an outlined ellipse centered at (cx, cy) with
// given width and height (diameters).
func ImageEllipse(dst draw.Image, cx, cy, width, height int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	drawMidpointEllipse(img, cx, cy, width, height, c)
	return true
}

// ImageFilledEllipse draws a filled ellipse.
func ImageFilledEllipse(dst draw.Image, cx, cy, width, height int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	rx := width / 2
	ry := height / 2
	if rx <= 0 || ry <= 0 {
		if rx == 0 && ry == 0 {
			img.plotPixel(cx, cy, c)
		}
		return true
	}
	rxSq := int64(rx) * int64(rx)
	rySq := int64(ry) * int64(ry)
	for dy := -ry; dy <= ry; dy++ {
		k := rxSq - (rxSq*int64(dy)*int64(dy))/rySq
		if k < 0 {
			k = 0
		}
		dx := isqrt(k)
		for x := cx - dx; x <= cx+dx; x++ {
			img.plotPixel(x, cy+dy, c)
		}
	}
	return true
}

func drawMidpointEllipse(img *Image, cx, cy, w, h int, c Color) {
	rx := w / 2
	ry := h / 2
	if rx <= 0 || ry <= 0 {
		if rx == 0 && ry == 0 {
			img.plotThick(cx, cy, c)
		}
		return
	}
	x, y := 0, ry
	rxSq := int64(rx) * int64(rx)
	rySq := int64(ry) * int64(ry)
	px := int64(0)
	py := 2 * rxSq * int64(y)

	plotFour := func(x, y int) {
		img.plotThick(cx+x, cy+y, c)
		img.plotThick(cx-x, cy+y, c)
		img.plotThick(cx+x, cy-y, c)
		img.plotThick(cx-x, cy-y, c)
	}

	p := rySq - rxSq*int64(ry) + rxSq/4
	for px < py {
		plotFour(x, y)
		x++
		px += 2 * rySq
		if p < 0 {
			p += rySq + px
		} else {
			y--
			py -= 2 * rxSq
			p += rySq + px - py
		}
	}

	p = rySq*int64(2*x+1)*int64(2*x+1)/4 + rxSq*(int64(y)-1)*(int64(y)-1) - rxSq*rySq
	for y >= 0 {
		plotFour(x, y)
		y--
		py -= 2 * rxSq
		if p > 0 {
			p += rxSq - py
		} else {
			x++
			px += 2 * rySq
			p += rxSq - py + px
		}
	}
}

// --- arc ---

// ImageArc draws an arc (section of an ellipse outline) from start to
// end degrees. Angles are measured clockwise from 3 o'clock.
func ImageArc(dst draw.Image, cx, cy, w, h, start, end int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	rx := float64(w) / 2
	ry := float64(h) / 2
	if rx <= 0 && ry <= 0 {
		img.plotThick(cx, cy, c)
		return true
	}

	a1 := float64(start) * math.Pi / 180
	a2 := float64(end) * math.Pi / 180
	if a2 < a1 {
		a2 += 2 * math.Pi
	}

	step := 0.5 / math.Max(rx, ry)
	lastX, lastY := math.MinInt, math.MinInt
	for a := a1; a <= a2; a += step {
		x := cx + int(math.Round(rx*math.Cos(a)))
		y := cy + int(math.Round(ry*math.Sin(a)))
		if x != lastX || y != lastY {
			img.plotThick(x, y, c)
			lastX, lastY = x, y
		}
	}
	x := cx + int(math.Round(rx*math.Cos(a2)))
	y := cy + int(math.Round(ry*math.Sin(a2)))
	if x != lastX || y != lastY {
		img.plotThick(x, y, c)
	}
	return true
}

// ImageFilledArc constants matching PHP gd.
const (
	ImgArcPie    = 0
	ImgArcChord  = 1
	ImgArcNoFill = 2
	ImgArcEdged  = 4
)

// ImageFilledArc draws a filled arc (pie slice or chord).
func ImageFilledArc(dst draw.Image, cx, cy, w, h, start, end int, c Color, style int) bool {
	if dst == nil {
		return false
	}
	rx := float64(w) / 2
	ry := float64(h) / 2

	a1 := float64(start) * math.Pi / 180
	a2 := float64(end) * math.Pi / 180
	if a2 < a1 {
		a2 += 2 * math.Pi
	}

	doFill := style&ImgArcNoFill == 0
	isChord := style&ImgArcChord != 0

	if doFill {
		// Collect arc boundary points and fill via scanline.
		var pts []image.Point
		if !isChord {
			pts = append(pts, image.Point{X: cx, Y: cy})
		}
		step := 0.5 / math.Max(math.Max(rx, ry), 1)
		lastX, lastY := math.MinInt, math.MinInt
		for a := a1; a <= a2; a += step {
			x := cx + int(math.Round(rx*math.Cos(a)))
			y := cy + int(math.Round(ry*math.Sin(a)))
			if x != lastX || y != lastY {
				pts = append(pts, image.Point{X: x, Y: y})
				lastX, lastY = x, y
			}
		}
		ex := cx + int(math.Round(rx*math.Cos(a2)))
		ey := cy + int(math.Round(ry*math.Sin(a2)))
		if ex != lastX || ey != lastY {
			pts = append(pts, image.Point{X: ex, Y: ey})
		}
		if len(pts) >= 3 {
			ImageFilledPolygon(dst, pts, c)
		}
	}

	// Draw outline edge.
	if !doFill || style&ImgArcEdged != 0 {
		ImageArc(dst, cx, cy, w, h, start, end, c)
		sx := cx + int(math.Round(rx*math.Cos(a1)))
		sy := cy + int(math.Round(ry*math.Sin(a1)))
		ex := cx + int(math.Round(rx*math.Cos(a2)))
		ey := cy + int(math.Round(ry*math.Sin(a2)))
		if isChord {
			ImageLine(dst, sx, sy, ex, ey, c)
		} else {
			ImageLine(dst, cx, cy, sx, sy, c)
			ImageLine(dst, cx, cy, ex, ey, c)
		}
	}
	return true
}

// --- flood fill ---

// ImageFill performs a flood fill starting at (x, y), replacing all
// connected pixels of the same color as (x, y) with c. Pass
// [ColorTiled] to paint with the tile set via [ImageSetTile].
func ImageFill(dst draw.Image, x, y int, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	clip := img.clipRect()
	if !(image.Point{X: x, Y: y}).In(clip) {
		return false
	}
	target := ImageColorAt(img, x, y)
	useTile := c == ColorTiled
	if useTile && img.tile == nil {
		return false
	}
	if !useTile && target == c {
		return true
	}
	type span struct{ x, y int }
	stack := []span{{x, y}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if ImageColorAt(img, p.x, p.y) != target {
			continue
		}
		lx := p.x
		for lx-1 >= clip.Min.X && ImageColorAt(img, lx-1, p.y) == target {
			lx--
		}
		rx := p.x
		for rx+1 < clip.Max.X && ImageColorAt(img, rx+1, p.y) == target {
			rx++
		}
		if useTile {
			tileFillRect(img, image.Rect(lx, p.y, rx+1, p.y+1))
		} else {
			for xi := lx; xi <= rx; xi++ {
				img.writeColor(xi, p.y, c)
			}
		}
		for _, dy := range [2]int{-1, 1} {
			yy := p.y + dy
			if yy < clip.Min.Y || yy >= clip.Max.Y {
				continue
			}
			run := false
			for xi := lx; xi <= rx; xi++ {
				if ImageColorAt(img, xi, yy) == target {
					if !run {
						stack = append(stack, span{xi, yy})
						run = true
					}
				} else {
					run = false
				}
			}
		}
	}
	return true
}

// ImageFillToBorder performs a flood fill starting at (x, y), filling all
// connected pixels until the border color is reached.
func ImageFillToBorder(dst draw.Image, x, y int, border, c Color) bool {
	img := asImage(dst)
	if img == nil {
		return false
	}
	clip := img.clipRect()
	if !(image.Point{X: x, Y: y}).In(clip) {
		return false
	}
	start := ImageColorAt(img, x, y)
	if start == border || start == c {
		return true
	}
	type span struct{ x, y int }
	canFill := func(x, y int) bool {
		cc := ImageColorAt(img, x, y)
		return cc != border && cc != c
	}
	stack := []span{{x, y}}
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !canFill(p.x, p.y) {
			continue
		}
		lx := p.x
		for lx-1 >= clip.Min.X && canFill(lx-1, p.y) {
			lx--
		}
		rx := p.x
		for rx+1 < clip.Max.X && canFill(rx+1, p.y) {
			rx++
		}
		for xi := lx; xi <= rx; xi++ {
			img.writeColor(xi, p.y, c)
		}
		for _, dy := range [2]int{-1, 1} {
			yy := p.y + dy
			if yy < clip.Min.Y || yy >= clip.Max.Y {
				continue
			}
			run := false
			for xi := lx; xi <= rx; xi++ {
				if canFill(xi, yy) {
					if !run {
						stack = append(stack, span{xi, yy})
						run = true
					}
				} else {
					run = false
				}
			}
		}
	}
	return true
}

// --- helpers ---

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func isqrt(n int64) int {
	if n <= 0 {
		return 0
	}
	r := int(math.Sqrt(float64(n)))
	for int64(r+1)*int64(r+1) <= n {
		r++
	}
	for int64(r)*int64(r) > n {
		r--
	}
	return r
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

