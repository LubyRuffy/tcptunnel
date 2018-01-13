package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"

	"github.com/xtaci/smux"
)

type onConnection func(net.Conn)
type onStream func(*smux.Stream)

func ioCopy(dst io.ReadWriter, src io.ReadWriter) (err error) {
	buf := make([]byte, 32*1024)
	n := 0
	for {
		n, err = src.Read(buf)
		if n > 0 {
			if _, e := dst.Write(buf[0:n]); e != nil {
				return e
			}
		}
		if err != nil {
			return
		}
	}
}

func IoBind(dst io.ReadWriteCloser, src io.ReadWriteCloser, fn func(err interface{})) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("bind crashed %s", err)
		}
	}()
	e1 := make(chan interface{}, 1)
	e2 := make(chan interface{}, 1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("bind crashed %s", err)
			}
		}()
		//_, err := io.Copy(dst, src)
		err := ioCopy(dst, src)
		e1 <- err
	}()
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("bind crashed %s", err)
			}
		}()
		//_, err := io.Copy(src, dst)
		err := ioCopy(src, dst)
		e2 <- err
	}()
	var err interface{}
	select {
	case err = <-e1:
		//log.Printf("e1")
	case err = <-e2:
		//log.Printf("e2")
	}
	src.Close()
	dst.Close()

	fn(err)
}

func random(min int, max int) string {
	return fmt.Sprintf("%v", rand.Intn(max-min)+min)
}

// 通用TCP监听流程
func listenTCPServer(wg *sync.WaitGroup, addr string, fn onConnection) (err error) {
	defer func() {
		wg.Done()
	}()

	var listener net.Listener
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("bind server to %s ok, listenning...\n", addr)
	for {
		var conn net.Conn
		conn, err = listener.Accept()
		if err != nil {
			log.Printf("Accept error, %v", err)
		}

		go func(newconn net.Conn) {
			fn(newconn)
		}(conn)
	}
}

func recvReq(stream *smux.Stream) (req *http.Request, err error) {
	r := bufio.NewReader(stream)
	req, err = http.ReadRequest(r)
	return
}
