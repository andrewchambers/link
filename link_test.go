package link

import (
	"mako/serial/link/concurrentbuffer"
	"testing"
	"time"
)

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
