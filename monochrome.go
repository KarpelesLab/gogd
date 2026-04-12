package gogd

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// --- WBMP ---

// ImageWBMP writes img to w as a WBMP type-0 (uncompressed monochrome)
// file. If foreground is a valid gd color, pixels exactly matching it
// are encoded as the WBMP foreground (bit 0, "black"); all other pixels
// become background (bit 1, "white"). Pass [ColorNone] to use a
// luminance threshold (dark < 128 → foreground).
func ImageWBMP(img image.Image, w io.Writer, foreground Color) error {
	if img == nil {
		return errNilImage
	}
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("gogd: wbmp dims %dx%d", width, height)
	}
	bw := bufio.NewWriter(w)
	if err := bw.WriteByte(0); err != nil { // type
		return err
	}
	if err := bw.WriteByte(0); err != nil { // fixed header
		return err
	}
	if err := writeWBMPInt(bw, width); err != nil {
		return err
	}
	if err := writeWBMPInt(bw, height); err != nil {
		return err
	}
	bpr := (width + 7) / 8
	row := make([]byte, bpr)
	for y := 0; y < height; y++ {
		for i := range row {
			row[i] = 0
		}
		for x := 0; x < width; x++ {
			if !isForegroundPixel(img, b.Min.X+x, b.Min.Y+y, foreground) {
				row[x/8] |= 1 << uint(7-x%8)
			}
		}
		if _, err := bw.Write(row); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// ImageCreateFromWBMP decodes a WBMP image from r. Only type 0 is
// supported (the only widely-deployed variant).
func ImageCreateFromWBMP(r io.Reader) (*Image, error) {
	br := bufio.NewReader(r)
	typeByte, err := br.ReadByte()
	if err != nil {
		return nil, err
	}
	if typeByte != 0 {
		return nil, fmt.Errorf("gogd: unsupported wbmp type %d", typeByte)
	}
	if _, err := br.ReadByte(); err != nil {
		return nil, err
	}
	width, err := readWBMPInt(br)
	if err != nil {
		return nil, err
	}
	height, err := readWBMPInt(br)
	if err != nil {
		return nil, err
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("gogd: invalid wbmp dims %dx%d", width, height)
	}
	bpr := (width + 7) / 8
	data := make([]byte, bpr*height)
	if _, err := io.ReadFull(br, data); err != nil {
		return nil, err
	}
	img := ImageCreate(width, height)
	black := ImageColorAllocate(img, 0, 0, 0)
	white := ImageColorAllocate(img, 255, 255, 255)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			bit := data[y*bpr+x/8] & (1 << uint(7-x%8))
			if bit != 0 {
				ImageSetPixel(img, x, y, white)
			} else {
				ImageSetPixel(img, x, y, black)
			}
		}
	}
	return img, nil
}

// readWBMPInt reads a WAP-style multi-byte integer: each byte
// contributes 7 bits, the high bit flags "more bytes follow", big-endian.
func readWBMPInt(r io.ByteReader) (int, error) {
	v := 0
	for i := 0; i < 5; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		v = (v << 7) | int(b&0x7f)
		if b&0x80 == 0 {
			return v, nil
		}
	}
	return 0, errors.New("gogd: wbmp integer too long")
}

// writeWBMPInt emits a WAP-style multi-byte integer.
func writeWBMPInt(w io.ByteWriter, v int) error {
	if v < 0 {
		return fmt.Errorf("gogd: negative wbmp int %d", v)
	}
	if v == 0 {
		return w.WriteByte(0)
	}
	var groups [5]byte
	n := 0
	for v > 0 {
		groups[n] = byte(v & 0x7f)
		v >>= 7
		n++
	}
	for i := n - 1; i >= 0; i-- {
		b := groups[i]
		if i > 0 {
			b |= 0x80
		}
		if err := w.WriteByte(b); err != nil {
			return err
		}
	}
	return nil
}

// --- XBM ---

var xbmDefineRe = regexp.MustCompile(`#define\s+\w*_?(width|height)\s+(\d+)`)
var xbmHexByteRe = regexp.MustCompile(`0[xX][0-9a-fA-F]+`)

// ImageXBM writes img to w as an X Bitmap. Same foreground semantics as
// [ImageWBMP]: matching the foreground color yields the "on" bit, all
// other pixels are "off". Pass [ColorNone] to use a luminance threshold.
// name is used as the C identifier prefix (e.g. "foo" produces
// `foo_width`, `foo_height`, `foo_bits`); default "image".
func ImageXBM(img image.Image, w io.Writer, foreground Color, name string) error {
	if img == nil {
		return errNilImage
	}
	if name == "" {
		name = "image"
	}
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("gogd: xbm dims %dx%d", width, height)
	}

	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "#define %s_width %d\n", name, width)
	fmt.Fprintf(bw, "#define %s_height %d\n", name, height)
	fmt.Fprintf(bw, "static unsigned char %s_bits[] = {\n", name)

	bpr := (width + 7) / 8
	count := 0
	for y := 0; y < height; y++ {
		for bx := 0; bx < bpr; bx++ {
			var v byte
			for bit := 0; bit < 8; bit++ {
				x := bx*8 + bit
				if x >= width {
					break
				}
				if isForegroundPixel(img, b.Min.X+x, b.Min.Y+y, foreground) {
					v |= 1 << uint(bit)
				}
			}
			if count > 0 {
				bw.WriteString(",")
			}
			if count%12 == 0 {
				bw.WriteString("\n  ")
			} else {
				bw.WriteString(" ")
			}
			fmt.Fprintf(bw, "0x%02x", v)
			count++
		}
	}
	bw.WriteString(" };\n")
	return bw.Flush()
}

// ImageCreateFromXBM decodes an X Bitmap image from r.
func ImageCreateFromXBM(r io.Reader) (*Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	text := string(data)
	defines := xbmDefineRe.FindAllStringSubmatch(text, -1)
	var width, height int
	for _, m := range defines {
		n, _ := strconv.Atoi(m[2])
		switch m[1] {
		case "width":
			width = n
		case "height":
			height = n
		}
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("gogd: xbm missing dimensions (%dx%d)", width, height)
	}

	start := strings.Index(text, "{")
	stop := strings.LastIndex(text, "}")
	if start < 0 || stop < 0 || stop < start {
		return nil, errors.New("gogd: xbm missing byte array")
	}
	body := text[start+1 : stop]
	hexes := xbmHexByteRe.FindAllString(body, -1)
	bpr := (width + 7) / 8
	need := bpr * height
	if len(hexes) < need {
		return nil, fmt.Errorf("gogd: xbm data truncated (have %d bytes, need %d)", len(hexes), need)
	}
	bytes := make([]byte, need)
	for i := 0; i < need; i++ {
		v, err := strconv.ParseUint(hexes[i][2:], 16, 16)
		if err != nil {
			return nil, fmt.Errorf("gogd: xbm hex parse: %w", err)
		}
		bytes[i] = byte(v)
	}

	img := ImageCreate(width, height)
	white := ImageColorAllocate(img, 255, 255, 255)
	black := ImageColorAllocate(img, 0, 0, 0)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// XBM is LSB-first within each byte; 1 = foreground (black).
			bit := bytes[y*bpr+x/8] & (1 << uint(x%8))
			if bit != 0 {
				ImageSetPixel(img, x, y, black)
			} else {
				ImageSetPixel(img, x, y, white)
			}
		}
	}
	return img, nil
}

// --- shared ---

// isForegroundPixel reports whether the pixel at (x, y) should be
// treated as "foreground" (drawn) when serialising to a 1-bit format.
// If foreground is a concrete gd color (packed truecolor or palette
// index), only pixels matching it are foreground. If foreground is
// [ColorNone] (or any negative sentinel), luminance threshold is used.
func isForegroundPixel(img image.Image, x, y int, foreground Color) bool {
	if foreground >= 0 {
		return ImageColorAt(img, x, y) == foreground
	}
	nc := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
	lum := int(nc.R)*299 + int(nc.G)*587 + int(nc.B)*114
	return lum < 128*1000
}
