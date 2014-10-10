package link

import (
	"bytes"
	"testing"
)

type myBuff struct {
	b bytes.Buffer
}

func (b *myBuff) Close() error {
	return nil
}

func (b *myBuff) Read(p []byte) (n int, err error) {
	return b.b.Read(p)
}

func (b *myBuff) Write(p []byte) (n int, err error) {
	return b.b.Write(p)
}

func TestLinkPacketReading(t *testing.T) {

	var b myBuff

	m1 := linkMessage{}
	m1.Kind = 1
	m2 := linkMessage{}
	m2.Kind = 2

	d1, _ := encodeMessage(&m1)
	d2, _ := encodeMessage(&m2)

	b.Write(d1)
	b.Write(d2)

	l := CreateLink(&b)

	_m1 := <-l.messageIn
	_m2 := <-l.messageIn

	if m1.Kind != _m1.Kind {
		t.Fatal("message did not match...")
	}

	if m2.Kind != _m2.Kind {
		t.Fatal("message did not match...")
	}

	defer l.Close()

}

func TestLinkPacketWriting(t *testing.T) {

	var b myBuff

	m1 := linkMessage{}
	m1.Kind = 5

	l := CreateLink(&b)

	l.messageOut <- m1

	m, _ := decodeMessage(b.b.Bytes())

	if m.Kind != m1.Kind {
		t.Fatal("message did not match")
	}

	defer l.Close()

}
