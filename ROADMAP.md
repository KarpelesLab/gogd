# Roadmap

gogd implements PHP gd functions on top of Go's standard `image`
packages. Each milestone is a self-contained, releasable slice.

Legend: `[x]` done · `[ ]` not started · `[~]` in progress

---

## M1 — Foundation `[~]`

Image type, truecolor vs palette, color allocation, direct pixel access.

- [x] `imagecreatetruecolor` → `ImageCreateTrueColor` (backs onto `image.NewNRGBA`)
- [x] `imagecreate` → `ImageCreate` (backs onto `image.NewPaletted`)
- [x] `imagedestroy` → `ImageDestroy` (no-op, GC)
- [x] `imageistruecolor` → `ImageIsTrueColor`
- [x] `imagesx` → `ImageSX`
- [x] `imagesy` → `ImageSY`
- [x] `imagecolorallocate` / `imagecolorallocatealpha`
- [x] `imagecolordeallocate`
- [x] `imagecolorat`
- [x] `imagecolorsforindex`
- [x] `imagecolorstotal`
- [x] `imagecolorexact` / `imagecolorexactalpha`
- [x] `imagecolorclosest` / `imagecolorclosestalpha`
- [x] `imagecolorresolve` / `imagecolorresolvealpha`
- [x] `imagecolortransparent`
- [x] `imagesetpixel`
- [x] `imagealphablending`
- [x] `imagesavealpha`
- [x] `gd_info` → `GDInfo`
- [x] `imagetypes` → `ImageTypes` (returns 0 until M2)
- [ ] `imagecolorclosesthwb`
- [ ] `imagecolormatch`
- [ ] `imagecolorset`

## M2 — I/O

Decoders and encoders via stdlib + `golang.org/x/image`.

- [ ] `imagepng` / `imagecreatefrompng` → `image/png`
- [ ] `imagejpeg` / `imagecreatefromjpeg` → `image/jpeg`
- [ ] `imagegif` / `imagecreatefromgif` → `image/gif`
- [ ] `imagebmp` / `imagecreatefrombmp` → `golang.org/x/image/bmp`
- [ ] `imagewebp` / `imagecreatefromwebp` → `golang.org/x/image/webp` (enc custom)
- [ ] `imagecreatefromstring`
- [ ] `image_type_to_extension`
- [ ] `image_type_to_mime_type`
- [ ] `getimagesize` / `getimagesizefromstring`
- [ ] `imageinterlace`

## M3 — Drawing primitives

- [ ] `imageline`, `imagedashedline`
- [ ] `imagerectangle`, `imagefilledrectangle`
- [ ] `imagepolygon`, `imageopenpolygon`, `imagefilledpolygon`
- [ ] `imageellipse`, `imagefilledellipse`
- [ ] `imagearc`, `imagefilledarc`
- [ ] `imagefill`, `imagefilltoborder`
- [ ] `imagesetthickness`
- [ ] `imagesetstyle`, `imagesetbrush`, `imagesettile`
- [ ] `imageantialias`
- [ ] `imagesetclip`, `imagegetclip`

## M4 — Copy / transform / scale

- [ ] `imagecopy`, `imagecopymerge`, `imagecopymergegray` → `image/draw`
- [ ] `imagecopyresized`, `imagecopyresampled`, `imagescale` → `golang.org/x/image/draw`
- [ ] `imagerotate`, `imageflip`
- [ ] `imagecrop`, `imagecropauto`
- [ ] `imageaffine`, `imageaffinematrixget`, `imageaffinematrixconcat`
- [ ] `imagegetinterpolation`, `imagesetinterpolation`
- [ ] `imageresolution`

## M5 — Filters & color operations

- [ ] `imagefilter` (all `IMG_FILTER_*` modes)
- [ ] `imageconvolution`
- [ ] `imagegammacorrect`
- [ ] `imagelayereffect`
- [ ] `imagepalettecopy`
- [ ] `imagepalettetotruecolor`
- [ ] `imagetruecolortopalette` (median-cut)

## M6 — Text

- [ ] `imagestring`, `imagestringup`, `imagechar`, `imagecharup`
- [ ] `imagefontwidth`, `imagefontheight`, `imageloadfont` — via `golang.org/x/image/font/basicfont`
- [ ] `imagettftext`, `imagettfbbox`, `imagefttext`, `imageftbbox` — via `golang.org/x/image/font/opentype`

## M7 — Niche / low-priority

- [ ] `imagegd` / `imagecreatefromgd`
- [ ] `imagegd2` / `imagecreatefromgd2` / `imagecreatefromgd2part`
- [ ] `imagewbmp` / `imagecreatefromwbmp`
- [ ] `imagexbm` / `imagecreatefromxbm`
- [ ] `imagecreatefromxpm`
- [ ] `imageavif` / `imagecreatefromavif`
- [ ] `imagecreatefromtga`
- [ ] `iptcembed`, `iptcparse`

## Out of scope

- `image2wbmp`, `jpeg2wbmp`, `png2wbmp` — removed in PHP 8.0.
- `imagegrabscreen`, `imagegrabwindow` — Windows-only in PHP, requires OS
  integration that doesn't belong in a pure image library.
