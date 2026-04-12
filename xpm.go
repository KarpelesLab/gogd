package gogd

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ImageCreateFromXPM decodes an X Window System pixmap (XPM3). The
// format is a C source file containing a `char *` array whose first
// entry is the "w h ncolors cpp" header, followed by ncolors color
// definitions and h rows of pixel data.
func ImageCreateFromXPM(r io.Reader) (*Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	entries, err := xpmExtractStrings(string(data))
	if err != nil {
		return nil, err
	}
	if len(entries) < 1 {
		return nil, errors.New("gogd: xpm: no strings")
	}
	var w, h, nColors, cpp int
	if _, err := fmt.Sscanf(entries[0], "%d %d %d %d", &w, &h, &nColors, &cpp); err != nil {
		return nil, fmt.Errorf("gogd: xpm header parse: %w", err)
	}
	if w <= 0 || h <= 0 || nColors <= 0 || cpp <= 0 {
		return nil, fmt.Errorf("gogd: xpm: invalid header (%d %d %d %d)", w, h, nColors, cpp)
	}
	if len(entries) < 1+nColors+h {
		return nil, fmt.Errorf("gogd: xpm: expected %d strings, got %d", 1+nColors+h, len(entries))
	}

	colorMap := map[string]xpmEntry{}
	for i := 0; i < nColors; i++ {
		s := entries[1+i]
		if len(s) < cpp+2 {
			return nil, fmt.Errorf("gogd: xpm: color entry %d too short: %q", i, s)
		}
		key := s[:cpp]
		rest := strings.TrimSpace(s[cpp:])
		// Find the "c" visual type; XPM supports mono/greyscale/symbol too.
		e, err := parseXPMColor(rest)
		if err != nil {
			return nil, fmt.Errorf("gogd: xpm: colour %q: %w", rest, err)
		}
		colorMap[key] = e
	}

	img := ImageCreateTrueColor(w, h)
	ImageAlphaBlending(img, false)
	for y := 0; y < h; y++ {
		row := entries[1+nColors+y]
		if len(row) < w*cpp {
			return nil, fmt.Errorf("gogd: xpm: row %d too short (%d < %d)", y, len(row), w*cpp)
		}
		for x := 0; x < w; x++ {
			key := row[x*cpp : x*cpp+cpp]
			e, ok := colorMap[key]
			if !ok {
				continue
			}
			if e.none {
				// Transparent.
				img.nrgba.Pix[(y*w+x)*4+3] = 0
				continue
			}
			pix := img.nrgba.Pix[(y*w+x)*4:]
			pix[0], pix[1], pix[2], pix[3] = e.r, e.g, e.b, 255
		}
	}
	ImageAlphaBlending(img, true)
	return img, nil
}

// xpmExtractStrings pulls out every double-quoted string from the C
// source body between the outermost `{` and `}`. Escapes like \" and \\
// are honoured.
func xpmExtractStrings(src string) ([]string, error) {
	start := strings.Index(src, "{")
	stop := strings.LastIndex(src, "}")
	if start < 0 || stop < 0 || stop < start {
		return nil, errors.New("gogd: xpm: missing array body")
	}
	body := src[start+1 : stop]
	var out []string
	i := 0
	for i < len(body) {
		if body[i] != '"' {
			i++
			continue
		}
		i++
		var sb strings.Builder
		for i < len(body) && body[i] != '"' {
			if body[i] == '\\' && i+1 < len(body) {
				sb.WriteByte(body[i+1])
				i += 2
				continue
			}
			sb.WriteByte(body[i])
			i++
		}
		if i < len(body) {
			i++ // skip closing quote
		}
		out = append(out, sb.String())
	}
	return out, nil
}

type xpmEntry struct {
	r, g, b, a uint8
	none       bool
}

func parseXPMColor(s string) (xpmEntry, error) {
	fields := strings.Fields(s)
	// XPM allows multiple "visual key" blocks per color: "m mono_color c rgb_color ...".
	// Prefer "c" (color), then "g" (grayscale), then "m" (monochrome).
	cval := findXPMVisual(fields, "c")
	if cval == "" {
		cval = findXPMVisual(fields, "g")
	}
	if cval == "" {
		cval = findXPMVisual(fields, "m")
	}
	if cval == "" {
		return xpmEntry{}, errors.New("no usable visual")
	}
	if strings.EqualFold(cval, "none") {
		return xpmEntry{none: true}, nil
	}
	return parseXPMColorValue(cval)
}

// findXPMVisual scans the color definition tokens for the value
// associated with the given visual key (c, g, m, s). Returns the value
// or "" if not present. The visual value may itself be multi-word
// (e.g. a named color like "light gray"), so we stop at the next
// known key.
func findXPMVisual(fields []string, key string) string {
	knownKeys := map[string]bool{"c": true, "g": true, "g4": true, "m": true, "s": true}
	for i := 0; i < len(fields); i++ {
		if fields[i] != key {
			continue
		}
		var buf []string
		for j := i + 1; j < len(fields); j++ {
			if knownKeys[fields[j]] {
				break
			}
			buf = append(buf, fields[j])
		}
		return strings.Join(buf, " ")
	}
	return ""
}

func parseXPMColorValue(v string) (xpmEntry, error) {
	if strings.HasPrefix(v, "#") {
		hex := v[1:]
		// Handle #RGB, #RRGGBB, #RRRRGGGGBBBB.
		switch len(hex) {
		case 3:
			r, err1 := parseHexNibble(hex[0])
			g, err2 := parseHexNibble(hex[1])
			b, err3 := parseHexNibble(hex[2])
			if err := firstErr(err1, err2, err3); err != nil {
				return xpmEntry{}, err
			}
			return xpmEntry{r: r * 17, g: g * 17, b: b * 17}, nil
		case 6:
			r, err1 := parseHexByte(hex[0:2])
			g, err2 := parseHexByte(hex[2:4])
			b, err3 := parseHexByte(hex[4:6])
			if err := firstErr(err1, err2, err3); err != nil {
				return xpmEntry{}, err
			}
			return xpmEntry{r: r, g: g, b: b}, nil
		case 12:
			r, err1 := parseHexByte(hex[0:2])
			g, err2 := parseHexByte(hex[4:6])
			b, err3 := parseHexByte(hex[8:10])
			if err := firstErr(err1, err2, err3); err != nil {
				return xpmEntry{}, err
			}
			return xpmEntry{r: r, g: g, b: b}, nil
		}
		return xpmEntry{}, fmt.Errorf("unsupported hex length %d", len(hex))
	}
	if named, ok := xpmNamedColors[strings.ToLower(v)]; ok {
		return xpmEntry{r: named[0], g: named[1], b: named[2]}, nil
	}
	return xpmEntry{}, fmt.Errorf("unknown color %q", v)
}

func parseHexNibble(b byte) (uint8, error) {
	n, err := strconv.ParseUint(string(b), 16, 8)
	return uint8(n), err
}

func parseHexByte(s string) (uint8, error) {
	n, err := strconv.ParseUint(s, 16, 8)
	return uint8(n), err
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// xpmNamedColors covers the small set of named colors commonly used in
// XPM files (a subset of X11's rgb.txt). Not exhaustive, but enough for
// the vast majority of real-world icons.
var xpmNamedColors = map[string][3]uint8{
	"black":   {0, 0, 0},
	"white":   {255, 255, 255},
	"red":     {255, 0, 0},
	"green":   {0, 128, 0},
	"blue":    {0, 0, 255},
	"yellow":  {255, 255, 0},
	"cyan":    {0, 255, 255},
	"magenta": {255, 0, 255},
	"gray":    {190, 190, 190},
	"grey":    {190, 190, 190},
	"none":    {0, 0, 0},
}
