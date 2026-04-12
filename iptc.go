package gogd

import (
	"encoding/binary"
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
