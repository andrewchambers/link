package main

import (
	"fmt"
	"mako/serial/link"
	"os"
)

func main() {
	l := link.CreateLink(os.Stdin, os.Stdout)

	for {
		con, err := l.Listen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "listen failed %s\n", err)
			return
		}
		fmt.Fprintln(os.Stderr, "got connection!")

		for {
			_, err = con.Write([]byte("hello\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "write %s", err)
				break
			}
		}

		con.Close()
	}
}
