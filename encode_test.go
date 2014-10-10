package link

import (
	"testing"
)

func TestEncodeEncDec(t *testing.T) {
	m := linkMessage{}
	data, err := encodeMessage(&m)
	if err != nil {
		t.Fatal("encoding failed... %v",err)
	}
	
	m2, err := decodeMessage(data)
	if err != nil {
	    t.Fatal("decoding failed... %v",err)
	}
	
	if m.Kind != m2.Kind {
	    t.Fatal("kinds do not match...")
	}
    
}
