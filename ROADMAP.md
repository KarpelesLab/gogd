# Roadmap

gogd implements PHP gd functions on top of Go's standard `image`
packages. Each milestone is a self-contained, releasable slice.

Legend: `[x]` done · `[ ]` not started · `[~]` in progress

---

## M1 — Foundation `[x]`

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

## M2 — I/O `[~]`

Decoders and encoders via stdlib + `golang.org/x/image`.

- [x] `imagepng` / `imagecreatefrompng` → `image/png`
- [x] `imagejpeg` / `imagecreatefromjpeg` → `image/jpeg`
- [x] `imagegif` / `imagecreatefromgif` → `image/gif`
- [x] `imagebmp` / `imagecreatefrombmp` → `golang.org/x/image/bmp`
- [~] `imagecreatefromwebp` → `golang.org/x/image/webp` — decode only; encoding still TODO (no pure-Go encoder in x/image)
- [x] `imagecreatefromstring`
- [x] `image_type_to_extension`
- [x] `image_type_to_mime_type`
- [x] `getimagesize` / `getimagesizefromstring`
- [ ] `imageinterlace`

## M3 — Drawing primitives `[~]`

- [x] `imageline`, `imagedashedline`
- [x] `imagerectangle`, `imagefilledrectangle`
- [x] `imagepolygon`, `imageopenpolygon`, `imagefilledpolygon`
- [x] `imageellipse`, `imagefilledellipse`
- [x] `imagearc`, `imagefilledarc`
- [x] `imagefill`, `imagefilltoborder`
- [x] `imagesetthickness`
- [ ] `imagesetstyle`, `imagesetbrush`, `imagesettile`
- [x] `imageantialias` (flag only; actual AA not yet implemented)
- [x] `imagesetclip`, `imagegetclip`

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
