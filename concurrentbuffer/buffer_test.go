package concurrentbuffer

import (
    "testing"
    "fmt"
    "bufio"
    "time"
)


func TestBuffer (t *testing.T) {
   
    buff := New(8)
    
    n,err := buff.Write([]byte("hello"))
    if n != 5 || err != nil {
        t.Fatal("failed write.")
    }
    
    n,err = buff.Write([]byte("hello"))
    
    if err == nil {
        t.Error("buffer should be full")
    }
    
    
    data := make([]byte,5)
    
    n,err = buff.Read(data)
    
    if err != nil { 
        t.Fatal(n,err)
    }
    
    if n != 5 || data[0] != 'h' || data[4] != 'o' {
        t.Fatal("read failed.",n)
    }
    
}

func TestUnevenReadWrite (t *testing.T) {
   
    buff := New(32)
    
    n,err := buff.Write([]byte { 0,1,2,3,4 })
    if n != 5 || err != nil {
        t.Fatal("failed write.")
    }
    
    n,err = buff.Write([]byte{ 5,6,7,8 })
    if n != 4 || err != nil {
        t.Fatal("failed write.")
    }
    
    data1 := make([]byte,7)
    data2 := make([]byte,2)
    n,err = buff.Read(data1)
    if err != nil {
        t.Fatal("bad read")
    }
    
    
    n,err = buff.Read(data2)
    if err != nil {
        t.Fatal("bad read")
    }
    
    for idx := range data1 {
        if data1[idx] != byte(idx) {
            t.Fatal("bad val", data1)
        } 
    }
    
    for idx := range data2 {
        if data2[idx] != byte(idx) + 7 {
            t.Fatal("bad val" , data2)
        } 
    }
    
}

func TestBufferConcurrent (t *testing.T) {
   
    buff := New(1024)
    
    go func() {
        for i := 0 ; i < 32 ; i++ {
            buff.Write([]byte(fmt.Sprintf("foo %d\n",i)))
        }
    }()
    
    go func() {
        b := bufio.NewReader(buff)
        for i := 0 ; i < 32 ; i++ {
            s,err := b.ReadString('\n')
            if err != nil {
                t.Fatal(err)
            }
            exp := fmt.Sprintf("foo %d\n",i)
            if s != exp {
                t.Fatalf("lines do not match")
            }
            
        }
    }()
    
}

func TestBufferClose (t *testing.T) {
   
    buff := New(1024)
    
    signal := make(chan struct{})
    
    go func() {
        n,err := buff.Read(make([]byte,16))
        if n != 0 || err == nil {
            t.Fatal("read should have failed.")
        }
        signal <- struct{}{}
    }()
    
    buff.Close()
    
    select {
        case <- signal:
        case <- time.After(5 * time.Second):
            t.Fatal("time out")
    }
    
}
