package device

import (
	"bytes"
	"testing"
)

func TestDerEncodeShortForm(t *testing.T) {
	got := derEncode(0x30, []byte{0x01, 0x02, 0x03})
	want := []byte{0x30, 0x03, 0x01, 0x02, 0x03}
	if !bytes.Equal(got, want) {
		t.Fatalf("derEncode short = %X, want %X", got, want)
	}
}

func TestDerEncodeLongFormTag(t *testing.T) {
	got := derEncode(0x5F37, []byte{0xAA})
	want := []byte{0x5F, 0x37, 0x01, 0xAA}
	if !bytes.Equal(got, want) {
		t.Fatalf("derEncode long tag = %X, want %X", got, want)
	}
	// Two-byte constructed tag used by every ES10 request.
	got = derEncode(0xBF38, nil)
	want = []byte{0xBF, 0x38, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("derEncode BF38 = %X, want %X", got, want)
	}
}

func TestDerEncodeLongFormLength(t *testing.T) {
	// 200 bytes forces an 0x81 long-form length.
	got := derEncode(0x30, make([]byte, 200))
	if got[0] != 0x30 || got[1] != 0x81 || got[2] != 200 {
		t.Fatalf("derEncode 200-byte length header = %X", got[:3])
	}
	// 1000 bytes forces an 0x82 long-form length.
	got = derEncode(0x30, make([]byte, 1000))
	if got[0] != 0x30 || got[1] != 0x82 || got[2] != 0x03 || got[3] != 0xE8 {
		t.Fatalf("derEncode 1000-byte length header = %X", got[:4])
	}
}

// The encoder must round-trip through the existing decoder (derParse).
func TestDerRoundTrip(t *testing.T) {
	inner := derConstruct(0xA0,
		derEncode(0x80, []byte{0x11, 0x22}),
		derEncode(0x5A, []byte{0x98, 0x10}),
	)
	outer := derConstruct(0xBF38, derEncode(0x30, inner), derEncode(0x04, []byte{0xFF}))

	nodes := derParse(outer)
	if len(nodes) != 1 || nodes[0].tag != 0xBF38 {
		t.Fatalf("expected single BF38 root, got %#v", nodes)
	}
	seq := derValue(nodes[0].children, 0x30)
	if seq == nil {
		t.Fatalf("missing 0x30 child")
	}
	if got := derFindValue(seq, 0x80); !bytes.Equal(got, []byte{0x11, 0x22}) {
		t.Fatalf("nested 0x80 = %X", got)
	}
	if got := derValue(nodes[0].children, 0x04); !bytes.Equal(got, []byte{0xFF}) {
		t.Fatalf("0x04 = %X", got)
	}
}

func TestDerElementAt(t *testing.T) {
	// BF38 { 30 03 010203 , 04 02 AABB }
	buf := derConstruct(0xBF38, derEncode(0x30, []byte{1, 2, 3}), derEncode(0x04, []byte{0xAA, 0xBB}))
	tag, headerLen, totalLen, err := derElementAt(buf, 0)
	if err != nil {
		t.Fatalf("derElementAt: %v", err)
	}
	if tag != 0xBF38 || headerLen != 3 || totalLen != len(buf) {
		t.Fatalf("root: tag=%X header=%d total=%d (len=%d)", tag, headerLen, totalLen, len(buf))
	}
	// First child starts right after the root header.
	tag, headerLen, totalLen, err = derElementAt(buf, 3)
	if err != nil || tag != 0x30 || headerLen != 2 || totalLen != 5 {
		t.Fatalf("child: tag=%X header=%d total=%d err=%v", tag, headerLen, totalLen, err)
	}
}

func TestUnwrapDER(t *testing.T) {
	// Already wrapped in 5F37 → returns inner value.
	wrapped := derEncode(0x5F37, []byte{0x01, 0x02})
	if got := unwrapDER(wrapped, 0x5F37); !bytes.Equal(got, []byte{0x01, 0x02}) {
		t.Fatalf("unwrapDER wrapped = %X", got)
	}
	// Bare value → returned unchanged.
	bare := []byte{0x09, 0x08}
	if got := unwrapDER(bare, 0x5F37); !bytes.Equal(got, bare) {
		t.Fatalf("unwrapDER bare = %X", got)
	}
}
