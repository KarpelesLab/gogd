package gogd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// IPTCParse decodes an IPTC IIM block (typically extracted from the
// Photoshop resource block of a JPEG APP13 marker). Each IPTC dataset
// is keyed "<record>#<ddd>" (three-digit dataset id) as PHP does, and
// the value is the slice of all entries seen for that key. Returns
// nil when no IPTC markers are found.
func IPTCParse(data []byte) map[string][]string {
	out := map[string][]string{}
	i := 0
	for i < len(data) {
		if data[i] != 0x1c {
			i++
			continue
		}
		if i+4 >= len(data) {
			break
		}
		record := data[i+1]
		dataset := data[i+2]
		length := 0
		lenFieldLen := 2
		if data[i+3]&0x80 == 0 {
			length = int(binary.BigEndian.Uint16(data[i+3 : i+5]))
		} else {
			// Extended length: low 7 bits say how many bytes follow
			// for the real length. Typical values are 4 or 8.
			lenFieldLen = int(data[i+3] & 0x7f)
			if lenFieldLen <= 0 || lenFieldLen > 8 || i+4+lenFieldLen > len(data) {
				break
			}
			for j := 0; j < lenFieldLen; j++ {
				length = length<<8 | int(data[i+4+j])
			}
			lenFieldLen++ // include the flag byte itself
		}
		off := i + 3 + lenFieldLen
		if off+length > len(data) {
			break
		}
		key := fmt.Sprintf("%d#%03d", record, dataset)
		out[key] = append(out[key], string(data[off:off+length]))
		i = off + length
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// IPTCEmbed injects an IPTC IIM data block into the APP13 marker of a
// JPEG file, using the Photoshop 3.0 / 8BIM 0x0404 convention most
// image-handling tools expect. Any existing APP13 marker in the input
// is replaced; all other segments (APP0/APP1/etc.) are preserved in
// order. Returns the modified JPEG bytes.
func IPTCEmbed(iptcData, jpegData []byte) ([]byte, error) {
	if len(jpegData) < 2 || jpegData[0] != 0xFF || jpegData[1] != 0xD8 {
		return nil, errors.New("gogd: iptcembed: input is not a JPEG (missing SOI)")
	}

	// Build the APP13 payload.
	var seg bytes.Buffer
	seg.WriteString("Photoshop 3.0\x00")
	seg.WriteString("8BIM")
	seg.WriteByte(0x04)
	seg.WriteByte(0x04)
	// Pascal-string resource name: length 0, padded to even total.
	seg.WriteByte(0x00)
	seg.WriteByte(0x00)
	if err := binary.Write(&seg, binary.BigEndian, uint32(len(iptcData))); err != nil {
		return nil, err
	}
	seg.Write(iptcData)
	if seg.Len()%2 != 0 {
		seg.WriteByte(0)
	}
	payload := seg.Bytes()
	if len(payload)+2 > 0xFFFF {
		return nil, fmt.Errorf("gogd: iptcembed: IPTC block too large (%d bytes)", len(payload))
	}

	var out bytes.Buffer
	out.Write(jpegData[:2]) // SOI

	i := 2
	appendedApp13 := false
	for i < len(jpegData) {
		if jpegData[i] != 0xFF {
			// Start-of-scan data or malformed — copy the rest verbatim.
			out.Write(jpegData[i:])
			break
		}
		// Skip fill bytes.
		j := i
		for j < len(jpegData) && jpegData[j] == 0xFF {
			j++
		}
		if j >= len(jpegData) {
			out.Write(jpegData[i:])
			break
		}
		marker := jpegData[j]
		segStart := i
		headerEnd := j + 1
		// Markers without length payload.
		switch {
		case marker == 0xD8, marker == 0xD9, marker == 0x01,
			(marker >= 0xD0 && marker <= 0xD7):
			out.Write(jpegData[segStart:headerEnd])
			i = headerEnd
			continue
		}
		if headerEnd+1 >= len(jpegData) {
			return nil, errors.New("gogd: iptcembed: truncated JPEG")
		}
		segLen := int(binary.BigEndian.Uint16(jpegData[headerEnd : headerEnd+2]))
		segEnd := headerEnd + segLen
		if segEnd > len(jpegData) {
			return nil, errors.New("gogd: iptcembed: segment length overruns file")
		}

		if marker == 0xED {
			// Drop existing APP13 — we'll emit our own below.
			i = segEnd
			continue
		}
		out.Write(jpegData[segStart:segEnd])
		i = segEnd

		// Insert our APP13 right after the first APP0 (or any first APPn).
		if !appendedApp13 && marker >= 0xE0 && marker <= 0xEF {
			writeAPP13(&out, payload)
			appendedApp13 = true
		}

		if marker == 0xDA { // Start of scan: image data follows to EOI.
			out.Write(jpegData[i:])
			break
		}
	}
	if !appendedApp13 {
		// No APPn segment present (unusual) — insert just after SOI,
		// which means rewriting the prefix we already emitted.
		var prefixed bytes.Buffer
		prefixed.Write(out.Bytes()[:2])
		writeAPP13(&prefixed, payload)
		prefixed.Write(out.Bytes()[2:])
		return prefixed.Bytes(), nil
	}
	return out.Bytes(), nil
}

func writeAPP13(w *bytes.Buffer, payload []byte) {
	w.WriteByte(0xFF)
	w.WriteByte(0xED)
	binary.Write(w, binary.BigEndian, uint16(len(payload)+2))
	w.Write(payload)
}
