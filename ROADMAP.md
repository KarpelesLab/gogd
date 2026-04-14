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
- [x] `imagecolorclosesthwb`
- [x] `imagecolormatch`
- [x] `imagecolorset` (landed in M5)

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
- [x] `imagesetbrush` (+ `ColorBrushed` stamping in ImageLine)
- [x] `imagesettile` (+ `ColorTiled` in filled rectangle and flood fill)
- [x] `imageantialias` — wires to Xiaolin Wu's algorithm in ImageLine
  (and transitively in polygons). Axis-aligned lines and thick lines
  still use Bresenham; thick AA is pending.
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

- [x] `imagefilter` — all IMG_FILTER_* modes (NEGATE, GRAYSCALE,
  BRIGHTNESS, CONTRAST, COLORIZE, EDGEDETECT, EMBOSS, GAUSSIAN_BLUR,
  SELECTIVE_BLUR, MEAN_REMOVAL, SMOOTH, PIXELATE, SCATTER)
- [x] `imageconvolution` — 3×3 convolution matrix with divisor + offset
- [x] `imagegammacorrect`
- [x] `imagelayereffect` — REPLACE/ALPHABLEND/NORMAL supported; OVERLAY/
  MULTIPLY accepted but not yet distinct
- [x] `imagecolorset`, `imagecolorsetalpha`
- [x] `imagepalettecopy`
- [x] `imagepalettetotruecolor`
- [x] `imagetruecolortopalette` — exact match when unique colours fit;
  classic median-cut otherwise
- [x] `imagecolormatch`
- [x] `imagecolorclosesthwb`

## M6 — Text `[~]`

- [x] `imagestring`, `imagestringup`, `imagechar`, `imagecharup` (all fontIDs map to `basicfont.Face7x13` for now)
- [x] `imagefontwidth`, `imagefontheight` (return PHP's reported dims for font IDs 1–5)
- [~] `imageloadfont` stub — gd's `.gd` font-file format not yet parsed
- [x] `imagettftext`, `imagettfbbox`, `imagefttext`, `imageftbbox` via `golang.org/x/image/font/opentype`; non-zero angle renders via an off-screen rotate-and-paste

## M7 — Niche / low-priority `[~]`

- [x] `imagegd` / `imagecreatefromgd` — libgd v1 format, truecolor + palette
- [~] `imagecreatefromgd2` — raw and zlib-compressed chunk formats (truecolor + palette). `imagegd2` encoder and `imagecreatefromgd2part` still pending.
- [x] `imagewbmp` / `imagecreatefromwbmp`
- [x] `imagexbm` / `imagecreatefromxbm`
- [x] `imagecreatefromxpm` — XPM3 including hex, short-hex, and common named colors
- [ ] `imageavif` / `imagecreatefromavif` — no pure-Go codec available
- [x] `imagecreatefromtga` — uncompressed + RLE, 24/32-bit truecolor, grayscale, and colormapped (types 1 & 9) with 15/16/24/32-bit color maps
- [x] `iptcparse`, `iptcembed`

## Out of scope

- `image2wbmp`, `jpeg2wbmp`, `png2wbmp` — removed in PHP 8.0.
- `imagegrabscreen`, `imagegrabwindow` — Windows-only in PHP, requires OS
  integration that doesn't belong in a pure image library.
