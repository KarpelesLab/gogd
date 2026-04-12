# gogd

A native Go implementation of the image operations exposed by PHP's
[gd extension](https://www.php.net/manual/en/ref.image.php), built on top of
Go's standard `image`, `image/color`, and `image/draw` packages.

**No cgo. No libgd.** Just Go.

## Status

Early development. See [ROADMAP.md](ROADMAP.md) for the milestone plan and
per-function status. Through **M3 — Drawing primitives**: image creation,
I/O (PNG/JPEG/GIF/BMP/WebP decode), color allocation, pixel access, lines,
rectangles, polygons, ellipses, arcs, flood fill, thickness and clipping.

## Install

```sh
go get github.com/KarpelesLab/gogd
```

## Example

```go
package main

import "github.com/KarpelesLab/gogd"

func main() {
    img := gogd.ImageCreateTrueColor(100, 100)
    red := gogd.ImageColorAllocate(img, 255, 0, 0)
    gogd.ImageSetPixel(img, 50, 50, red)
    // Encoders (PNG, JPEG, GIF) land in M2.
}
```

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

## Scope

gogd targets the modern PHP 8+ gd surface. Functions removed upstream
(`image2wbmp`, `jpeg2wbmp`, `png2wbmp`) are not implemented.

## License

MIT — see [LICENSE](LICENSE). Copyright © 2026 Karpelès Lab Inc.
