package link

import (
	"bytes"
	"errors"
	"mako/serial/link/concurrentbuffer"
	"sync"
	"testing"
	"time"
)

type myBuff struct {
	m      sync.Mutex
	b      bytes.Buffer
	closed bool
}

func (b *myBuff) Close() error {
	b.m.Lock()
	defer b.m.Unlock()
	b.closed = true
	return nil
}

func (b *myBuff) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()

	if b.closed {
		return 0, errors.New("buff closed")
	}

	return b.b.Read(p)
}

func (b *myBuff) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()

	if b.closed {
		return 0, errors.New("buff closed")
	}

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

	_m1, err := l.Read(make(chan struct{}),-1)
	if err != nil {
		t.Fatal(err)
	}
	_m2, err := l.Read(make(chan struct{}),-1)
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

func TestLinkDial2(t *testing.T) {

	b1 := concurrentbuffer.New(0)
	b2 := concurrentbuffer.New(0)

	l1 := CreateLink(b1, b2)
	l2 := CreateLink(b2, b1)

	done := make(chan struct{})

	// server
	go func() {
		_, err := l1.Listen()
		if err != nil {
			t.Fatal("listen failed...")
		}
		done <- struct{}{}
	}()

	// client
	go func() {
		_, err := l2.Dial()
		if err != nil {
			t.Fatal("listen failed...")
		}
		done <- struct{}{}
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// do nothing.
		case <-time.After(1 * time.Second):
			t.Fatal("timeout...")
		}
	}
}

func TestLinkChat(t *testing.T) {

	b1 := concurrentbuffer.New(0)
	b2 := concurrentbuffer.New(0)

	l1 := CreateLink(b1, b2)
	l2 := CreateLink(b2, b1)

	done := make(chan struct{})

	// server
	go func() {
		con, err := l1.Listen()
		if err != nil {
			t.Fatal("listen failed...")
		}
		_, err = con.Write([]byte("olleh"))
		if err != nil {
			t.Error(err)
		}
		_, err = con.Write([]byte(" world!"))
		if err != nil {
			t.Error(err)
		}

		buff := make([]byte, 5)

		n, err := con.Read(buff)
		if n != 5 || err != nil {
			t.Error("failed...", n, err)
		}
		if string(buff) != "hello" {
			t.Error("fail.", string(buff))
		}
		n, err = con.Read(buff)
		if n != 5 || err != nil {
			t.Error("failed...", n, err)
		}
		if string(buff) != " worl" {
			t.Error("fail.", string(buff))
		}
		n, err = con.Read(buff)
		if n != 2 || err != nil {
			t.Error("failed...", n, err)
		}
		if string(buff[0:2]) != "d!" {
			t.Error("fail.", string(buff))
		}

		done <- struct{}{}
	}()

	// client
	go func() {
		con, err := l2.Dial()
		if err != nil {
			t.Fatal("listen failed...")
		}

		_, err = con.Write([]byte("hello"))
		if err != nil {
			t.Error(err)
		}
		_, err = con.Write([]byte(" world!"))
		if err != nil {
			t.Error(err)
		}

		buff := make([]byte, 5)

		n, err := con.Read(buff)
		if n != 5 || err != nil {
			t.Error("failed...", n, err)
		}
		if string(buff) != "olleh" {
			t.Error("fail.", string(buff))
		}
		n, err = con.Read(buff)
		if n != 5 || err != nil {
			t.Error("failed...", n, err)
		}
		if string(buff) != " worl" {
			t.Error("fail.", string(buff))
		}
		n, err = con.Read(buff)
		if n != 2 || err != nil {
			t.Error("failed...", n, err)
		}
		if string(buff[0:2]) != "d!" {
			t.Error("fail.", string(buff))
		}

		done <- struct{}{}
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
			// do nothing.
		case <-time.After(1 * time.Second):
			t.Fatal("timeout...")
		}
	}
}
