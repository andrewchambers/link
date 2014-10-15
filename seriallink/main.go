package main

import (
	"fmt"
	"net"
	"io"
	"os"
	"mako/serial/link"
)

func help() {
    fmt.Println("seriallink provides a reliable link over lossy serial ports")
    fmt.Println("on a mako run: seriallink ")
    os.Exit(0)
}


func proxy(conn1 ,conn2 net.Conn) {
    defer conn1.Close()
    defer conn2.Close()
    
    done := make(chan struct{},2)
    
    go func () {
        io.Copy(conn1,conn2)
        done <- struct{}{}
    } ()
    
    go func() {
        io.Copy(conn2,conn1)
        done <- struct{}{}
    } ()
    
    <- done
    
}

func tcp2link() error {
    l,err := net.Listen("tcp","127.0.0.1:")
    if err != nil {
        return err
    }
    defer l.Close()
    fmt.Printf("listening on %s\n",l.Addr())
    
    linkconn,err := net.Dial("tcp","127.0.0.1:8000")
    if err != nil {
        return fmt.Errorf("dialing remote end of link failed. %s",err)
    }
    defer linkconn.Close()
    link := link.CreateLink(linkconn,linkconn)
        
    for {
        // Only accept one connection at a time.
        conn1,err := l.Accept()
        if err != nil {
            return err
        }
        fmt.Printf("incoming connection from %s. \n",conn1.RemoteAddr())
        conn2,err := link.Dial()
        if err != nil {
            conn1.Close()
            if link.IsDown() {
                return err
            } else {
                fmt.Printf("dialing on link failed. %s\n",err)
                continue
            }
        }
        proxy(conn1,conn2)
        fmt.Println("connection closed.\n")
    }
}

func handle_link2tcp(lconn net.Conn) {
    tcpconn,err := net.Dial("tcp","127.0.0.1:22")
    if err != nil {
        lconn.Close()
        fmt.Fprintf(os.Stderr,"failed to dial tcp conn. %s\n",err)
        return
    }
    proxy(tcpconn,lconn)    
}

func link2tcp() error {
    l := link.CreateLink(os.Stdin,os.Stdout)
    defer l.Close()
    for {
        lconn,err := l.Accept()
        if err != nil {
            return err
        }
        handle_link2tcp(lconn)
    }
}

func main() {
    args := os.Args
    if len(args) < 2 {
        help()
    }
    
    switch args[1] {
        case "tcp2link":
            err := tcp2link()
            if err != nil {
                fmt.Println("failed to listen for connections.",err)
                os.Exit(1)
            }
        case "link2tcp":
            err := link2tcp()
            if err != nil {
                fmt.Println("failed to listen for connections.",err)
                os.Exit(1)
            }
        default:
            fmt.Println("unknown mode! %s",args[1])
            os.Exit(1)
    }

}
