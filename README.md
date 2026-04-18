# gogd

A native Go implementation of the image operations exposed by PHP's
[gd extension](https://www.php.net/manual/en/ref.image.php), built on top of
Go's standard `image`, `image/color`, and `image/draw` packages.

**No cgo. No libgd.** Just Go.

## Install

```sh
go get github.com/KarpelesLab/gogd
```

## Example

```go
package main

import (
    "os"
    "github.com/KarpelesLab/gogd"
)

func main() {
    img := gogd.ImageCreateTrueColor(200, 120)
    red := gogd.ImageColorAllocate(img, 255, 0, 0)
    gogd.ImageAntialias(img, true)
    gogd.ImageLine(img, 10, 10, 190, 110, red)
    gogd.ImageFilledEllipse(img, 100, 60, 80, 50, red)
    f, _ := os.Create("out.png")
    defer f.Close()
    gogd.ImagePNG(img, f)
}
```

## Features

- **I/O.** PNG / JPEG / GIF / BMP / WebP / AVIF round-trip
  ([gowebp](https://github.com/KarpelesLab/gowebp) for WebP,
  [goavif](https://github.com/KarpelesLab/goavif) for AVIF — both pure
  Go, lossy + lossless), WBMP, XBM, XPM, TGA (uncompressed + RLE,
  truecolor + grayscale + colormapped), libgd's own `.gd` v1 format
  round-trip and GD2 read.
- **Drawing.** Lines (Bresenham or Xiaolin Wu antialiased), dashed lines,
  rectangles, polygons, ellipses, arcs (pie and chord), flood fill, fill-
  to-border. `imagesetstyle` / `imagesetbrush` / `imagesettile` honoured.
- **Transforms.** Copy, merge, gray merge, resized / resampled / scaled
  (nearest, bilinear, bicubic via `golang.org/x/image/draw`), rotate,
  flip, crop, auto-crop, affine (with matrix helpers).
- **Filters.** Every `IMG_FILTER_*` mode (negate, grayscale, brightness,
  contrast, colorize, edge detect, emboss, Gaussian and selective blur,
  mean removal, smooth, pixelate, scatter). Plus generic 3×3 convolution
  and gamma correction.
- **Color.** Full palette API (allocate, closest, closest-HWB, exact,
  resolve, set, match), truecolor ↔ palette conversion with median-cut
  quantisation.
- **Text.** Bitmap text (`imagestring` family) and TrueType text
  (`imagettftext`, `imagettfbbox`) at any angle via
  `golang.org/x/image/font`.
- **Metadata.** `iptcparse` and `iptcembed` for JPEG APP13 blocks.

## Design

- `gogd.Image` implements `image.Image` and `draw.Image`, so it can be
  passed to anything in Go's image ecosystem (`image/draw`, `image/png`,
  third-party resamplers, etc.) without adapters.
- The reverse is also true: gogd functions accept stdlib images
  directly. `ImageSetPixel`, `ImageLine`, `ImageFilledRectangle`,
  `ImageFilter`, `ImageCopy`, `ImageRotate`, `ImagePNG`, etc. take
  `image.Image` / `draw.Image`, so you can call gogd operations on a
  `*image.NRGBA`, `*image.RGBA`, or `*image.Paletted` without wrapping.
  gd state (alpha blending, clip, thickness) defaults sensibly for
  non-gogd images; use `*gogd.Image` when you need those controls.
- Truecolor images are backed by `*image.NRGBA`; palette images by
  `*image.Paletted`.
- gd's 7-bit alpha channel (0 = opaque, 127 = transparent) is translated
  to and from Go's 8-bit alpha on the boundary — you pass gd values and
  the stdlib sees conventional NRGBA.
- Function names mirror PHP gd (`ImageCreateTrueColor`,
  `ImageColorAllocate`, `ImageSetPixel`, …) so porting PHP code is
  mechanical. Idiomatic Go shortcuts (`img.Width()`, `img.Height()`,
  `img.IsTrueColor()`) are also provided.

## Known limitations

These PHP gd functions have partial or no support — most are blocked on
things outside the library's control:

- **`imageinterlace`.** Flag is stored, but Go's stdlib PNG and JPEG
  encoders don't expose an interlace / progressive knob.
- **`imageloadfont`.** libgd's custom `.gd` font-file format isn't parsed;
  all five built-in bitmap font IDs render through a single
  `basicfont.Face7x13` face for now.
- **`imagegd2` encoder and `imagecreatefromgd2part`.** GD2 *read* is
  fully supported (raw + zlib-compressed, truecolor + palette). Writing
  GD2 and reading a sub-rect are not yet implemented.
- **Antialiased thick lines and axis-aligned lines.** AA uses Xiaolin Wu
  for diagonal lines at thickness 1. Thick AA and AA curves on arcs /
  ellipses fall back to plain Bresenham.
- **`imagelayereffect` OVERLAY / MULTIPLY.** REPLACE, ALPHABLEND, and
  NORMAL are honoured; the other two modes are accepted but currently
  no-ops.
- **Colormapped TGA at non-byte-aligned indices.** 8- and 16-bit indices
  are supported; custom odd widths are not.

Explicitly out of scope (removed upstream or OS-specific):

- `image2wbmp`, `jpeg2wbmp`, `png2wbmp` — removed in PHP 8.0.
- `imagegrabscreen`, `imagegrabwindow` — Windows-only screen capture,
  requires OS integration that doesn't belong in a pure image library.

## License

MIT — see [LICENSE](LICENSE). Copyright © 2026 Karpelès Lab Inc.
