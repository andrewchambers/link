package link

import (
	"testing"
)

func TestEncDec(t *testing.T) {
	m := linkMessage{}
	m.Kind = 127
	m.Data = []byte{1, 2, 3, 4, 5, 6}

	data, err := encodeMessage(&m)
	if err != nil {
		t.Fatalf("encoding failed... %v", err)
	}
	m2, err := decodeMessage(data)
	if err != nil {
		t.Fatalf("decoding failed... %v", err)
	}

	if m.Kind != m2.Kind {
		t.Fatal("kinds do not match...")
	}

	if m2.Data[3] != 4 {
		t.Fatal("data does not match...")
	}

	data[5]++

	m2, err = decodeMessage(data)
	if err == nil {
		t.Fatal("checksum should have failed")
	}
}
