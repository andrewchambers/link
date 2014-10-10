package link

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
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

type LinkSession struct {
	state int
	link  *Link
}

type Link struct {
        
	wrc        io.ReadWriteCloser
	messageIn  <-chan linkMessage
	messageOut chan<- linkMessage

	// This channel is closed on shutdown...
	closeInputOnce,closeOutputOnce sync.Once
	// Closed on shutdown, don't send anything to this.
	inputdown chan struct{}
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

func CreateLink(wrc io.ReadWriteCloser) *Link {
	in := make(chan linkMessage)
	out := make(chan linkMessage)
	ret := &Link{
		wrc:        wrc,
		messageIn:  in,
		messageOut: out,
		inputdown:   make(chan struct{}),
		outputdown:   make(chan struct{}),
	}
	go ret.readMessages(in)
	go ret.writeMessages(out)
	return ret
}

func (link *Link) Read(timeout time.Duration) (linkMessage,error) {
	
	var timeoutChan <-chan time.Time = make(chan time.Time)
	
	if timeout > 0 {
	    ticker := time.NewTicker(timeout)
	    timeoutChan = ticker.C
	    defer ticker.Stop()
	}
	
	select {
	case m := <- link.messageIn:
		return m,nil
	case <-timeoutChan:
	    return linkMessage{},fmt.Errorf("timeout")
	case <-link.inputdown:
		return linkMessage{},fmt.Errorf("link down.")
	}
}

func (link *Link) Write(timeout time.Duration,m linkMessage) error {
	
	var timeoutChan <-chan time.Time = make(chan time.Time)
	
	if timeout > 0 {
	    ticker := time.NewTicker(timeout)
	    timeoutChan = ticker.C
	    defer ticker.Stop()
	}
	
	select {
	case link.messageOut <- m:
		return nil
	case <-timeoutChan:
	    return fmt.Errorf("timeout")
	case <-link.outputdown:
		return fmt.Errorf("link down.")
	}
}

func (link *Link) readMessages(ch chan<- linkMessage) {
	reader := bufio.NewReader(link.wrc)
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
			_, err = link.wrc.Write(encoded)
			if err != nil {
				return
			}
		case <-link.outputdown:
			return
		}
	}
}

func (link *Link) Listen() (net.Conn, error) {

	
	for {
	    m,err  := link.Read(-1)
	    if err != nil {
	        return nil,err
	    }
	    if m.Kind == CONNECT {
	        ack := linkMessage{}
	        ack.Kind = ACK
	        err = link.Write(-1,ack)
	        if err != nil {
	            return nil,err
	        }
	        err = link.Write(-1,ack)
	        if err != nil {
	            return nil,err
	        }
	        connected := false
	        
	        select {
	            case ackack := <- link.messageIn:
	                if ackack.Kind == ACKACK {
	                    connected = true
	                }
	            case <- time.After(1 * time.Second):
	                connected = false
	        }
	        if connected {
	            break 
	        }
	    }
	}
	
	ret := &LinkSession{}
	ret.link = link
	
	return ret, nil
}

func (link *Link) Dial() (net.Conn, error) {
    connected := false
	for i := 0 ; i < 3 ; i++ {
	    m := linkMessage{}
	    m.Kind = CONNECT
	    link.messageOut <- m
        
        select {
            case ack := <- link.messageIn:
                if ack.Kind == ACK {
                    ackack:= linkMessage{}
	                ackack.Kind = ACKACK
	                link.messageOut <- ackack
	                link.messageOut <- ackack
                    connected = true
                }
            case <- time.After(1 * time.Second):
                connected = false
        }
        if connected {
            break 
        }
    }
    
    if !connected {
        return nil,fmt.Errorf("failed to establish connection.")
    }

	ret := &LinkSession{}
	ret.link = link	
	return ret, nil
}

type dummyLinkAddr struct{}

func (*dummyLinkAddr) Network() string {
	return "link"
}

func (*dummyLinkAddr) String() string {
	return "link"
}

func (s *LinkSession) Read(b []byte) (n int, err error) {
	panic("unimplemented")
}

func (s *LinkSession) Write(b []byte) (n int, err error) {
	panic("unimplemented")
}

func (s *LinkSession) Close() error {
	panic("unimplemented")
}

func (s *LinkSession) LocalAddr() net.Addr {
	return &dummyLinkAddr{}
}

func (s *LinkSession) RemoteAddr() net.Addr {
	return &dummyLinkAddr{}
}

func (s *LinkSession) SetDeadline(t time.Time) error {
	return fmt.Errorf("unimplemented")
}

func (s *LinkSession) SetReadDeadline(t time.Time) error {
	return fmt.Errorf("unimplemented")
}

func (s *LinkSession) SetWriteDeadline(t time.Time) error {
	return fmt.Errorf("unimplemented")
}
