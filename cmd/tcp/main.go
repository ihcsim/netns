package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	network       = "tcp"
	maxPacketSize = 4096
	port          = 4078
)

var timeout = time.Second * 5

func main() {
	addr := &net.TCPAddr{Port: port}
	listener, err := net.ListenTCP(network, addr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer listener.Close()
	log.Printf("listening at %s (%s)...", listener.Addr(), listener.Addr().Network())

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		defer conn.Close()

		now := time.Now()
		deadline := now.Add(timeout)
		if err := conn.SetDeadline(deadline); err != nil {
			log.Println(err)
			// next connection has no deadline
		}

		data := make([]byte, maxPacketSize)
		nRecv, err := conn.Read(data)
		if err != nil {
			var e *net.OpError
			if errors.As(err, &e) && e.Timeout() {
				// if timeout, restart loop
				continue
			}

			if errors.Is(err, io.EOF) {
				continue
			}

			log.Println(err)
			continue
		}

		// trim empty spaces in block
		end := strings.IndexByte(string(data), '\x00')
		if end != -1 {
			data = data[:end]
		}
		if nRecv == 0 {
			log.Printf("no data received [size_bytes=%d]", nRecv)
			continue
		}
		log.Printf("received: %q (size_bytes=%d)", strings.TrimSpace(string(data)), nRecv)

		// send response
		resp := []byte(fmt.Sprintf("[%s] %s", now.Format("2006-01-02 15:04:05"), data))
		if _, err := conn.Write(resp); err != nil {
			log.Println(err)
		}
	}
}
