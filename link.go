package link

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"mako/serial/link/concurrentbuffer"
	"net"
	"sync"
	"time"
)

type linkSessionState int

const (
	CONNECTED = iota
	DISCONNECTED
)

type LinkSession struct {
	readBuff io.ReadWriteCloser
	// Must be held while sending data.
	writeLock sync.Mutex

	ackChannel chan uint

	keepAliveChannel chan struct{}

	closeOnce sync.Once
	closed    chan struct{}

	state          int
	curSeqnum      uint
	expectedSeqnum uint
	link           *Link
}

var ErrTimeout = fmt.Errorf("timeout")

type Link struct {
	r          io.ReadCloser
	w          io.WriteCloser
	messageIn  <-chan linkMessage
	messageOut chan<- linkMessage

	// This channel is closed on shutdown...
	closeInputOnce, closeOutputOnce sync.Once
	// Closed on shutdown, don't send anything to this.
	inputdown  chan struct{}
	outputdown chan struct{}
}

func (link *Link) Close() {
	link.closeInput()
	link.closeOutput()
}

func (link *Link) closeInput() {
	f := func() {
		close(link.inputdown)
	}
	link.closeInputOnce.Do(f)
}

func (link *Link) closeOutput() {
	f := func() {
		close(link.outputdown)
	}
	link.closeOutputOnce.Do(f)
}

func CreateLink(r io.ReadCloser, w io.WriteCloser) *Link {
	in := make(chan linkMessage)
	out := make(chan linkMessage)
	ret := &Link{
		r:          r,
		w:          w,
		messageIn:  in,
		messageOut: out,
		inputdown:  make(chan struct{}),
		outputdown: make(chan struct{}),
	}
	go ret.readMessages(in)
	go ret.writeMessages(out)
	return ret
}

func (link *Link) Read(cancel chan struct{},timeout time.Duration) (linkMessage, error) {

	var timeoutChan <-chan time.Time = make(chan time.Time)

	if timeout > 0 {
		ticker := time.NewTicker(timeout)
		timeoutChan = ticker.C
		defer ticker.Stop()
	}

	select {
	case m := <-link.messageIn:
		return m, nil
	case <-timeoutChan:
		return linkMessage{}, ErrTimeout
	case <-link.inputdown:
		return linkMessage{}, fmt.Errorf("link down.")
	case <-cancel:
		return linkMessage{}, fmt.Errorf("read cancelled")
	}
}

func (link *Link) Write(cancel chan struct{},timeout time.Duration, m linkMessage) error {

	var timeoutChan <-chan time.Time

	if timeout > 0 {
		ticker := time.NewTicker(timeout)
		timeoutChan = ticker.C
		defer ticker.Stop()
	} else {
		timeoutChan = make(chan time.Time)
	}

	select {
	case link.messageOut <- m:
		return nil
	case <-timeoutChan:
		return ErrTimeout
	case <-cancel:
	    return fmt.Errorf("write cancelled")
	case <-link.outputdown:
		return fmt.Errorf("link down.")
	}
}

func (link *Link) readMessages(ch chan<- linkMessage) {
	reader := bufio.NewReader(link.r)
	defer link.closeInput()
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
		case <-link.inputdown:
			return
		}
	}
}

func (link *Link) writeMessages(ch <-chan linkMessage) {
	defer link.closeOutput()
	for {
		select {
		case m := <-ch:
			encoded, err := encodeMessage(&m)
			if err != nil {
				return
			}
			_, err = link.w.Write(encoded)
			if err != nil {
				return
			}
		case <-link.outputdown:
			return
		}
	}
}

func (link *Link) Listen() (net.Conn, error) {
    cancel := make(chan struct{})
	for {
		m, err := link.Read(cancel,-1)
		if err != nil {
			return nil, err
		}
		if m.Kind == CONNECT {
			ack := linkMessage{}
			ack.Kind = ACK
			err = link.Write(cancel,-1, ack)
			if err != nil {
				return nil, err
			}
			ackack, err := link.Read(cancel,1 * time.Second)
			if err != nil {
				if err == ErrTimeout {
					continue
				}
				return nil, err
			}
			if ackack.Kind == ACKACK {
				break
			}
		}
	}

	ret := newSession(link)

	return ret, nil
}

func (link *Link) Dial() (net.Conn, error) {
    cancel := make(chan struct{})
	connected := false
	for i := 0; i < 5; i++ {
		m := linkMessage{}
		m.Kind = CONNECT
		link.Write(cancel,-1, m)

		ack, err := link.Read(cancel,1 * time.Second)
		if err != nil {
			if err == ErrTimeout {
				continue
			}
			return nil, err
		}
		if ack.Kind == ACK {
			ackack := linkMessage{}
			ackack.Kind = ACKACK
			err := link.Write(cancel,-1, ackack)
			if err != nil {
				return nil, err
			}
			err = link.Write(cancel,-1, ackack)
			if err != nil {
				return nil, err
			}
			connected = true
		}
		if connected {
			break
		}
	}
	if !connected {
		return nil, fmt.Errorf("failed to establish connection.")
	}
	ret := newSession(link)
	return ret, nil
}

func newSession(link *Link) *LinkSession {
	ret := &LinkSession{}
	ret.link = link
	// Max buff is 1 meg for now.
	ret.readBuff = concurrentbuffer.New(1024 * 1024)
	ret.ackChannel = make(chan uint)
	ret.keepAliveChannel = make(chan struct{})
	ret.closed = make(chan struct{})
	go ret.handleMessages()
	go ret.handleTimeout()
	go ret.handlePings()
	return ret
}

type dummyLinkAddr struct{}

func (*dummyLinkAddr) Network() string {
	return "link"
}

func (*dummyLinkAddr) String() string {
	return "link"
}

func (s *LinkSession) sendAck(seqnum uint) error {
	ackmessage := linkMessage{}
	ackmessage.Kind = ACK
	ackmessage.Seqnum = seqnum
	err := s.link.Write(s.closed,-1, ackmessage)
	if err != nil {
		s.Close()
	}
	return err
}

func (s *LinkSession) sendData(seqnum uint, data []byte) error {
	d := linkMessage{}
	d.Kind = DATA
	d.Seqnum = seqnum
	d.Data = data
	err := s.link.Write(s.closed,-1, d)
	if err != nil {
		s.Close()
	}
	return err
}

func (s *LinkSession) handlePings() {
	defer s.Close()
	p := linkMessage{}
	p.Kind = PING
	for {
		if s.isClosed() {
			return
		}
		time.Sleep(1 * time.Second)
		err := s.link.Write(s.closed,-1, p)
		if err != nil {
			return
		}
	}
}

func (s *LinkSession) handleTimeout() {
	defer s.Close()

	duration := 5 * time.Second

	timer := time.NewTimer(duration)
	defer timer.Stop() // Might not be needed....

	for {
		select {
		case <-s.keepAliveChannel:
			timer.Reset(duration)
		case <-timer.C:
			return
		case <-s.closed:
			return
		}

	}

}

func (s *LinkSession) handleMessages() {
	defer s.Close()
	for {
		m, err := s.link.Read(s.closed,-1)
		if err != nil {
			return
		}
		switch m.Kind {
		case PING:
			s.keepAliveChannel <- struct{}{}
		case ACK:
			s.keepAliveChannel <- struct{}{}
			//XXX dropped ack packets slow everything down considerably.
			// refactor somehow?
			select {
			case s.ackChannel <- m.Seqnum:
			case <-time.After(1 * time.Millisecond):
				// Noone is listening, discard ack.
				// They will have to try again.
			}
		case DATA:
			s.keepAliveChannel <- struct{}{}
			switch {
			case m.Seqnum == s.expectedSeqnum:
				_, err := s.readBuff.Write(m.Data)
				if err == concurrentbuffer.BufferFull {
					// Drop packet.
				} else if err != nil {
					return
				} else if err == nil {
					s.expectedSeqnum++
					err := s.sendAck(m.Seqnum)
					if err != nil {
						return
					}
				}
			case m.Seqnum < s.expectedSeqnum:
				err := s.sendAck(m.Seqnum)
				if err != nil {
					return
				}
			default:
				//nothing
			}
		default:
		}
	}
}

func (s *LinkSession) Read(b []byte) (int, error) {
	return s.readBuff.Read(b)
}

func (s *LinkSession) Write(b []byte) (int, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	seqnum := s.curSeqnum

	for {
		if s.isClosed() {
			return 0, errors.New("session closed")
		}
		err := s.sendData(seqnum, b)
		if err != nil {
			s.Close()
			return 0, err
		}
		select {
		case recievedSeqnum := <-s.ackChannel:
			if recievedSeqnum == seqnum {
				s.curSeqnum++
				return len(b), nil
			}
		case <-time.After(15 * time.Millisecond): // XXX make this based of round trip.
			//Resend via looping.
		case <-s.closed:
			//will quit on loop
		}
	}

}

func (s *LinkSession) Close() error {
	f := func() {
		close(s.closed)
		s.readBuff.Close()
	}
	s.closeOnce.Do(f)
	return nil
}

func (s *LinkSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *LinkSession) LocalAddr() net.Addr {
	return &dummyLinkAddr{}
}

func (s *LinkSession) RemoteAddr() net.Addr {
	return &dummyLinkAddr{}
}

func (s *LinkSession) SetDeadline(t time.Time) error {
	panic("unimplemented")
}

func (s *LinkSession) SetReadDeadline(t time.Time) error {
	panic("unimplemented")
}

func (s *LinkSession) SetWriteDeadline(t time.Time) error {
	panic("unimplemented")
}
