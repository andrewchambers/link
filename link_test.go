package link

import (
	"bytes"
	"sync"
	"testing"
)

type myBuff struct {
	m sync.Mutex
	b bytes.Buffer
}

func (b *myBuff) Close() error {
	return nil
}

func (b *myBuff) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *myBuff) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

func (b *myBuff) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Bytes()
}

func TestLinkPacketReading(t *testing.T) {

	var in, out myBuff

	m1 := linkMessage{}
	m1.Kind = 1
	m2 := linkMessage{}
	m2.Kind = 2

	d1, _ := encodeMessage(&m1)
	d2, _ := encodeMessage(&m2)

	in.Write(d1)
	in.Write(d2)

	l := CreateLink(&in, &out)

	_m1, err := l.Read(-1)
	if err != nil {
		t.Fatal(err)
	}
	_m2, err := l.Read(-1)
	if err != nil {
		t.Fatal(err)
	}

	if m1.Kind != _m1.Kind {
		t.Fatal("message did not match...")
	}

	if m2.Kind != _m2.Kind {
		t.Fatal("message did not match...")
	}

	defer l.Close()

}

func TestLinkPacketWriting(t *testing.T) {

	var in, out myBuff

	m1 := linkMessage{}
	m1.Kind = 5

	l := CreateLink(&in, &out)

	l.Write(-1, m1)

	m, err := decodeMessage(out.Bytes())

	if err != nil {
		t.Fatal(err)
	}

	if m.Kind != m1.Kind {
		t.Fatal("message did not match")
	}

	defer l.Close()

}

func TestLinkListen(t *testing.T) {

	var in, out myBuff

	m := linkMessage{}
	m.Kind = CONNECT

	d, _ := encodeMessage(&m)
	in.Write(d)

	m.Kind = ACKACK
	d, _ = encodeMessage(&m)
	in.Write(d)

	l := CreateLink(&in, &out)
	defer l.Close()

	_, err := l.Listen()
	if err != nil {
		t.Fatal(err)
	}

}

func TestLinkDial(t *testing.T) {

	var in, out myBuff

	m := linkMessage{}
	m.Kind = ACK

	d, _ := encodeMessage(&m)
	in.Write(d)

	l := CreateLink(&in, &out)
	defer l.Close()

	_, err := l.Dial()
	if err != nil {
		t.Fatal(err)
	}

}
