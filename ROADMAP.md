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
- [~] `imageinterlace` — flag plumbed; Go's stdlib encoders don't expose an interlace knob so the flag is stored only

## M3 — Drawing primitives `[~]`

- [x] `imageline`, `imagedashedline`
- [x] `imagerectangle`, `imagefilledrectangle`
- [x] `imagepolygon`, `imageopenpolygon`, `imagefilledpolygon`
- [x] `imageellipse`, `imagefilledellipse`
- [x] `imagearc`, `imagefilledarc`
- [x] `imagefill`, `imagefilltoborder`
- [x] `imagesetthickness`
- [x] `imagesetstyle` (+ `ColorStyled`/`ColorTransparent` sentinels)
- [~] `imagesetbrush` (+ `ColorBrushed` stamping in ImageLine); `imagesettile` stub only
- [x] `imageantialias` (flag only; actual AA not yet implemented)
- [x] `imagesetclip`, `imagegetclip`

## M4 — Copy / transform / scale `[x]`

- [x] `imagecopy`, `imagecopymerge`, `imagecopymergegray` → `image/draw`
- [x] `imagecopyresized`, `imagecopyresampled`, `imagescale` → `golang.org/x/image/draw`
- [x] `imagerotate`, `imageflip`
- [x] `imagecrop`
- [x] `imagecropauto` — modes: DEFAULT, TRANSPARENT, BLACK, WHITE, SIDES, THRESHOLD
- [x] `imageaffine`, `imageaffinematrixget`, `imageaffinematrixconcat`
- [x] `imagegetinterpolation`, `imagesetinterpolation`
- [x] `imageresolution` (plus a Go-style `ImageGetResolution` getter)

## M5 — Filters & color operations `[~]`

- [~] `imagefilter` — NEGATE, GRAYSCALE, BRIGHTNESS, CONTRAST, COLORIZE,
  EDGEDETECT, EMBOSS, GAUSSIAN_BLUR, MEAN_REMOVAL, SMOOTH, PIXELATE.
  SELECTIVE_BLUR and SCATTER not yet implemented.
- [x] `imageconvolution` — 3×3 convolution matrix with divisor + offset
- [x] `imagegammacorrect`
- [x] `imagelayereffect` — REPLACE/ALPHABLEND/NORMAL supported; OVERLAY/
  MULTIPLY accepted but not yet distinct
- [x] `imagecolorset`, `imagecolorsetalpha`
- [x] `imagepalettecopy`
- [x] `imagepalettetotruecolor`
- [x] `imagetruecolortopalette` — simple histogram quantizer (exact match
  when unique colours fit; top-N by frequency otherwise). Median-cut upgrade
  pending.
- [ ] `imagecolormatch`
- [ ] `imagecolorclosesthwb`

## M6 — Text `[~]`

- [x] `imagestring`, `imagestringup`, `imagechar`, `imagecharup` (all fontIDs map to `basicfont.Face7x13` for now)
- [x] `imagefontwidth`, `imagefontheight` (return PHP's reported dims for font IDs 1–5)
- [~] `imageloadfont` stub — gd's `.gd` font-file format not yet parsed
- [x] `imagettftext`, `imagettfbbox`, `imagefttext`, `imageftbbox` via `golang.org/x/image/font/opentype`; non-zero angle renders via an off-screen rotate-and-paste

## M7 — Niche / low-priority `[~]`

- [ ] `imagegd` / `imagecreatefromgd`
- [ ] `imagegd2` / `imagecreatefromgd2` / `imagecreatefromgd2part`
- [x] `imagewbmp` / `imagecreatefromwbmp`
- [x] `imagexbm` / `imagecreatefromxbm`
- [ ] `imagecreatefromxpm`
- [ ] `imageavif` / `imagecreatefromavif`
- [ ] `imagecreatefromtga`
- [ ] `iptcembed`, `iptcparse`

## Out of scope

- `image2wbmp`, `jpeg2wbmp`, `png2wbmp` — removed in PHP 8.0.
- `imagegrabscreen`, `imagegrabwindow` — Windows-only in PHP, requires OS
  integration that doesn't belong in a pure image library.
