package gogd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
)

// ImageCreateFromTGA decodes a Truevision Targa image from r. Supports
// uncompressed and RLE-compressed truecolor (image types 2 and 10) at
// 24 or 32 bits per pixel, plus grayscale (types 3 and 11). Colormapped
// variants (types 1 and 9) are not yet supported.
func ImageCreateFromTGA(r io.Reader) (*Image, error) {
	br := bufio.NewReader(r)
	var hdr [18]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return nil, err
	}
	idLen := int(hdr[0])
	colorMapType := int(hdr[1])
	imgType := int(hdr[2])
	mapFirst := int(binary.LittleEndian.Uint16(hdr[3:5]))
	mapLen := int(binary.LittleEndian.Uint16(hdr[5:7]))
	mapEntrySize := int(hdr[7])
	width := int(binary.LittleEndian.Uint16(hdr[12:14]))
	height := int(binary.LittleEndian.Uint16(hdr[14:16]))
	pixDepth := int(hdr[16])
	descriptor := int(hdr[17])
	topDown := descriptor&0x20 != 0
	rightToLeft := descriptor&0x10 != 0
	_ = mapFirst

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("gogd: invalid tga dims %dx%d", width, height)
	}
	switch imgType {
	case 1, 2, 3, 9, 10, 11:
	default:
		return nil, fmt.Errorf("gogd: unsupported tga image type %d", imgType)
	}
	if pixDepth%8 != 0 || pixDepth == 0 {
		return nil, fmt.Errorf("gogd: unsupported tga bit depth %d", pixDepth)
	}

	if idLen > 0 {
		if _, err := io.CopyN(io.Discard, br, int64(idLen)); err != nil {
			return nil, err
		}
	}

	var palette []color.NRGBA
	if colorMapType == 1 {
		if mapEntrySize%8 != 0 || mapEntrySize < 15 || mapEntrySize > 32 {
			return nil, fmt.Errorf("gogd: unsupported tga color-map entry size %d", mapEntrySize)
		}
		entryBytes := (mapEntrySize + 7) / 8
		mapBuf := make([]byte, mapLen*entryBytes)
		if _, err := io.ReadFull(br, mapBuf); err != nil {
			return nil, err
		}
		palette = make([]color.NRGBA, mapLen)
		for i := 0; i < mapLen; i++ {
			palette[i] = decodeTGAColor(mapBuf[i*entryBytes:i*entryBytes+entryBytes], mapEntrySize)
		}
	}

	bpp := pixDepth / 8
	pixels := make([]byte, width*height*bpp)
	if imgType == 9 || imgType == 10 || imgType == 11 {
		if err := readTGARLE(br, pixels, bpp); err != nil {
			return nil, err
		}
	} else {
		if _, err := io.ReadFull(br, pixels); err != nil {
			return nil, err
		}
	}

	img := ImageCreateTrueColor(width, height)
	ImageAlphaBlending(img, false)
	for sy := 0; sy < height; sy++ {
		dy := sy
		if !topDown {
			dy = height - 1 - sy
		}
		for sx := 0; sx < width; sx++ {
			dx := sx
			if rightToLeft {
				dx = width - 1 - sx
			}
			off := (sy*width + sx) * bpp
			var nc color.NRGBA
			switch imgType {
			case 2, 10:
				nc = color.NRGBA{
					B: pixels[off],
					G: pixels[off+1],
					R: pixels[off+2],
					A: 255,
				}
				if bpp >= 4 {
					nc.A = pixels[off+3]
				}
			case 3, 11:
				gr := pixels[off]
				nc = color.NRGBA{R: gr, G: gr, B: gr, A: 255}
			case 1, 9:
				idx := 0
				switch bpp {
				case 1:
					idx = int(pixels[off])
				case 2:
					idx = int(pixels[off]) | int(pixels[off+1])<<8
				default:
					return nil, fmt.Errorf("gogd: unsupported tga colormap index bpp %d", bpp)
				}
				if idx < 0 || idx >= len(palette) {
					continue
				}
				nc = palette[idx]
			}
			img.nrgba.SetNRGBA(dx, dy, nc)
		}
	}
	ImageAlphaBlending(img, true)
	return img, nil
}

// decodeTGAColor unpacks a TGA color-map entry. TGA stores entries in
// little-endian order; 15/16-bit entries use 5-5-5(-1) ARRRRRGG GGGBBBBB
// layout (bit 15 is attribute/alpha in the 16-bit form).
func decodeTGAColor(b []byte, bits int) color.NRGBA {
	switch bits {
	case 15, 16:
		v := uint16(b[0]) | uint16(b[1])<<8
		r := uint8((v >> 10) & 0x1f)
		g := uint8((v >> 5) & 0x1f)
		bl := uint8(v & 0x1f)
		return color.NRGBA{R: r * 8, G: g * 8, B: bl * 8, A: 255}
	case 24:
		return color.NRGBA{R: b[2], G: b[1], B: b[0], A: 255}
	case 32:
		return color.NRGBA{R: b[2], G: b[1], B: b[0], A: b[3]}
	}
	return color.NRGBA{A: 255}
}

// readTGARLE decodes a TGA RLE stream into out. Each packet is one
// header byte: high bit set means the next pixel is repeated
// (count = (header & 0x7f) + 1) times; high bit clear means the next
// (count) pixels are stored verbatim.
func readTGARLE(r io.Reader, out []byte, bpp int) error {
	n := 0
	total := len(out)
	var hdrByte [1]byte
	for n < total {
		if _, err := io.ReadFull(r, hdrByte[:]); err != nil {
			return err
		}
		count := int(hdrByte[0]&0x7f) + 1
		if hdrByte[0]&0x80 != 0 {
			pixel := make([]byte, bpp)
			if _, err := io.ReadFull(r, pixel); err != nil {
				return err
			}
			for i := 0; i < count; i++ {
				if n+bpp > total {
					return fmt.Errorf("gogd: tga rle overflow")
				}
				copy(out[n:n+bpp], pixel)
				n += bpp
			}
		} else {
			need := count * bpp
			if n+need > total {
				return fmt.Errorf("gogd: tga raw overflow")
			}
			if _, err := io.ReadFull(r, out[n:n+need]); err != nil {
				return err
			}
			n += need
		}
	}
	return nil
}
