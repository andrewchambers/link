package link

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
)

type messageKind int

type linkState int

const (
	UNINIT = iota
	LISTENING
	CONNECTING
	CONNECTED
	DISCONNECTED
)

type Link struct {
	state int

	wrc        io.ReadWriteCloser
	messageIn  <-chan linkMessage
	messageOut chan<- linkMessage

	// This channel is closed on shutdown...
	shutdownOnce sync.Once
	// Closed on shutdown, don't send anything to this.
	shutdown chan struct{}
}

func (link *Link) Close() {
	f := func() {
		close(link.shutdown)
	}
	link.shutdownOnce.Do(f)
}

func CreateLink(wrc io.ReadWriteCloser) *Link {
	in := make(chan linkMessage)
	out := make(chan linkMessage)
	ret := &Link{
		wrc:        wrc,
		messageIn:  in,
		messageOut: out,
		shutdown:   make(chan struct{}),
	}
	go ret.readMessages(in)
	go ret.writeMessages(out)
	return ret
}

func (link *Link) readMessages(ch chan<- linkMessage) {
	reader := bufio.NewReader(link.wrc)
	for {
		line, err := reader.ReadBytes('~')
		if err != nil {
			return
		}
		m, err := decodeMessage(line)
		if err != nil {
			continue
		}
		select {
		case ch <- m:
			// Sent...
		case <-link.shutdown:
			return
		}
	}
}
func (link *Link) writeMessages(ch <-chan linkMessage) {
	for {
		select {
		case m := <-ch:
			encoded, err := encodeMessage(&m)
			if err != nil {
				return
			}
			_, err = link.wrc.Write(encoded)
			if err != nil {
				return
			}
		case <-link.shutdown:
			return
		}
	}
}

func (link *Link) Listen() (net.Conn, error) {
	return nil, fmt.Errorf("unimplemented")
}
