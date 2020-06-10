package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	dataBlockSizeBytes = 4096
	network            = "udp"
	port               = 47733
)

var timeoutRead = time.Second * 5

func main() {
	laddr := &net.UDPAddr{Port: port}
	conn, err := net.ListenUDP(network, laddr)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer conn.Close()
	log.Printf("listening at %s (%s)...", conn.LocalAddr(), conn.LocalAddr().Network())

	for {
		// set r/w deadline to now+5 seconds
		now := time.Now()
		deadline := now.Add(timeoutRead)
		if err := conn.SetDeadline(deadline); err != nil {
			log.Printf("failed to set deadline. next read has no deadline: %s", err)
		}

		data := make([]byte, dataBlockSizeBytes)
		nRecv, addr, err := conn.ReadFrom(data)
		if err != nil {
			var e *net.OpError
			if errors.As(err, &e) && e.Timeout() {
				// if timeout, extend the deadline and try again
				deadline = time.Now().Add(timeoutRead)
				continue
			}

			log.Println(err)
			os.Exit(1)
		}

		// trim empty spaces in block
		end := strings.IndexByte(string(data), '\x00')
		if end != -1 {
			data = data[:end]
		}

		if nRecv == 0 {
			log.Printf("no data received [size_bytes=%d,src_addr=%s]", nRecv, addr)
			continue
		}
		log.Printf("received: %q (size_bytes=%d,src_addr=%s)", strings.TrimSpace(string(data)), nRecv, addr)

		// send response
		resp := fmt.Sprintf("[%s] %s", now.Format("2006-01-02 15:04:05"), data)
		if _, err := conn.WriteTo([]byte(resp), addr); err != nil {
			var e *net.OpError
			if errors.As(err, &e) && e.Timeout() {
				// if timeout, extend the deadline and try again
				deadline = time.Now().Add(timeoutRead)
				continue
			}

			log.Println(err)
			os.Exit(1)
		}
		log.Printf("replied with: %q (size_bytes=%d,dest_addr=%s)", strings.TrimSpace(string(resp)), nRecv, addr)
	}
}
