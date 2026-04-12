package gogd

import (
	"testing"
)

func buildIPTCBlock(entries ...[3]any) []byte {
	// each entry: [record, dataset, value string]
	var out []byte
	for _, e := range entries {
		record := e[0].(int)
		dataset := e[1].(int)
		val := []byte(e[2].(string))
		out = append(out, 0x1c, byte(record), byte(dataset))
		out = append(out, byte(len(val)>>8), byte(len(val)))
		out = append(out, val...)
	}
	return out
}

func TestIPTCParse(t *testing.T) {
	block := buildIPTCBlock(
		[3]any{2, 5, "A Photo"},
		[3]any{2, 25, "travel"},
		[3]any{2, 25, "sunset"},
	)
	m := IPTCParse(block)
	if m == nil {
		t.Fatal("parse returned nil")
	}
	if v := m["2#005"]; len(v) != 1 || v[0] != "A Photo" {
		t.Errorf("2#005 = %v", v)
	}
	kws := m["2#025"]
	if len(kws) != 2 || kws[0] != "travel" || kws[1] != "sunset" {
		t.Errorf("2#025 = %v", kws)
	}
}

func TestIPTCParseEmpty(t *testing.T) {
	if m := IPTCParse([]byte("no IPTC here")); m != nil {
		t.Errorf("expected nil, got %v", m)
	}
}

func TestIPTCParseTruncated(t *testing.T) {
	// Length says 100 bytes but we only have 3 after header.
	block := []byte{0x1c, 2, 5, 0x00, 100, 'a', 'b', 'c'}
	m := IPTCParse(block)
	if m != nil {
		t.Errorf("expected nil for truncated, got %v", m)
	}
}
