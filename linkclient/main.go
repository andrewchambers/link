package main

import (
	"fmt"
	"mako/serial/link"
	"os/exec"
)

func main() {

	cmd := exec.Command("linkserver")
	in, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("failed to create command stdin pipe")
		return
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("failed to create command stdout pipe")
		return
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println("failed to start linkserver")
		return
	}

	l := link.CreateLink(out, in)
	defer l.Close()
	con, err := l.Dial()
	if err != nil {
		fmt.Printf("dial failed: %s\n", err)
		return
	}
	defer con.Close()

	buff := make([]byte, 5)
	_, err = con.Read(buff)
	if err != nil {
		fmt.Printf("read failed: %s\n", err)
		return
	}

	fmt.Printf("%s\n", buff)
}
