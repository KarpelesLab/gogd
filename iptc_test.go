package gogd

import (
	"bytes"
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

func TestIPTCEmbedAddsAPP13(t *testing.T) {
	img := ImageCreateTrueColor(4, 4)
	var buf bytes.Buffer
	if err := ImageJPEG(img, &buf, 80); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	iptc := buildIPTCBlock([3]any{2, 5, "Hello"})
	out, err := IPTCEmbed(iptc, buf.Bytes())
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	// APP13 marker is 0xFF 0xED; must be present in result.
	if !bytes.Contains(out, []byte{0xFF, 0xED}) {
		t.Error("APP13 marker not present in output")
	}
	// The Photoshop identifier must be present.
	if !bytes.Contains(out, []byte("Photoshop 3.0\x00")) {
		t.Error("Photoshop identifier not present")
	}
	// The IPTC content should survive verbatim.
	if !bytes.Contains(out, iptc) {
		t.Error("IPTC payload not in output")
	}
	// Round-trip through IPTCParse: extract APP13 content and check.
	// Quick check: the "Hello" string must exist somewhere.
	if !bytes.Contains(out, []byte("Hello")) {
		t.Error("dataset value missing")
	}
}

func TestIPTCEmbedReplacesExistingAPP13(t *testing.T) {
	img := ImageCreateTrueColor(4, 4)
	var buf bytes.Buffer
	ImageJPEG(img, &buf, 80)
	first, err := IPTCEmbed(buildIPTCBlock([3]any{2, 5, "first"}), buf.Bytes())
	if err != nil {
		t.Fatalf("embed 1: %v", err)
	}
	second, err := IPTCEmbed(buildIPTCBlock([3]any{2, 5, "second"}), first)
	if err != nil {
		t.Fatalf("embed 2: %v", err)
	}
	if bytes.Contains(second, []byte("first")) {
		t.Error("old IPTC block should have been replaced")
	}
	if !bytes.Contains(second, []byte("second")) {
		t.Error("new IPTC block missing")
	}
}

func TestIPTCEmbedRejectsNonJPEG(t *testing.T) {
	if _, err := IPTCEmbed(nil, []byte{0x00, 0x01, 0x02}); err == nil {
		t.Error("expected error for non-JPEG input")
	}
}
