package link

import (
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
	for {
		var m linkMessage
		select {
		case ch <- m:
		case <-link.shutdown:
			panic("cancel")
		}
	}
}
func (link *Link) writeMessages(ch <-chan linkMessage) {
	for {
		select {
		case _ = <-ch:
		case <-link.shutdown:
			panic("cancel")
		}
	}
}

func (link *Link) Listen() (net.Conn, error) {
	return nil, fmt.Errorf("unimplemented")
}
