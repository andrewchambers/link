package concurrentbuffer

// This package implements a concurrent buffer.
// You can specify the max amount of data to be buffered, reads block until
// writes are ready.

import (
    "errors"
    "sync"
    "io"
)



type bufferedData struct {
    bytes []byte
    next *bufferedData
}

type concurrentBuffer struct {
    cond *sync.Cond
    d *bufferedData
    tail *bufferedData
    closed bool
    sz uint
    maxsz uint
}


// Return a new buffer.
// maxBuffering is the maximum number of bytes the buffer can store.
// The internal representation may take more space than this.
func New(maxBuffering uint ) io.ReadWriteCloser {
    ret := &concurrentBuffer{}
    ret.maxsz = maxBuffering
    ret.cond = sync.NewCond(&sync.Mutex{})
    return ret
}


func (b *concurrentBuffer) Read(p []byte) (int, error) {
    b.cond.L.Lock()
    for b.sz == 0 {
        if b.closed {
            b.cond.L.Unlock()
            return 0,errors.New("buffer closed")
        }        
        b.cond.Wait()
    }
    
    amntToRead := len(p)
    if b.sz < uint(amntToRead) {
        amntToRead = int(b.sz)
        if amntToRead < 0 {
            panic("overflow - is maxsz too large?")
        }
    }
    
    n := 0
    for n != amntToRead {
        nToCopy := amntToRead - n
        switch {
            case nToCopy >= len(b.d.bytes):
                for i := 0 ; i < len(b.d.bytes) ; i++  {
                    p[n + i] = b.d.bytes[i]
                }
                n += len(b.d.bytes)
                b.d = b.d.next
                if b.d == nil {
                    b.tail = nil
                }
            case nToCopy < len(b.d.bytes):
                for i := 0 ; i < nToCopy ; i++  {
                    p[n + i] = b.d.bytes[i]
                }
                b.d.bytes = b.d.bytes[nToCopy:]
                n += nToCopy
            default:
                panic("unreachable")
        }
    }
    
    b.sz -= n
    b.cond.L.Unlock()
    return n,nil
}


func (b *concurrentBuffer) Write(p []byte) (int, error) {
     
     b.cond.L.Lock()
     
     if b.sz + uint(len(p)) > b.maxsz {
        b.cond.L.Unlock()
        return 0, errors.New("buffer full")
     } 
     
     node := &bufferedData{}
     node.bytes = make([]byte,len(p))
     n := copy(node.bytes,p)
     if b.d == nil {
         if b.sz != 0 || b.tail != nil {
             panic("internal error")
         }
        b.d = node
        b.tail = node
     } else {
        b.tail.next = node
        b.tail = node
     }
     
     b.sz += uint(n)
     
     b.cond.L.Unlock()
     b.cond.Broadcast()
     
     return n,nil
}

func (b *concurrentBuffer) Close() error {
    b.cond.L.Lock()
    b.closed = true
    b.cond.L.Unlock()
    b.cond.Broadcast()
    return nil
}
