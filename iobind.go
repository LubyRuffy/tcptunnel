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

func handleRequest(conn net.Conn, outConn net.Conn, inReq *http.Request, remoteAddr string) (resp *http.Response, err error) {
	if inReq == nil {
		return
	}

	outReq := new(http.Request)
	*outReq = *inReq // includes shallow copies of maps, but we handle this in Director
	outReq.Host = remoteAddr
	err = outReq.Write(outConn)
	if err != nil {
		log.Printf("[http] %s - %s : %s", conn.RemoteAddr(), conn.LocalAddr(), err)
		return
	}

	resp, err = http.ReadResponse(bufio.NewReader(outConn), outReq)
	if err != nil {
		log.Printf("[http] %s - %s : %s", conn.RemoteAddr(), conn.LocalAddr(), err)
		return
	}
	return
}

func HTTPBind(conn net.Conn, outConn net.Conn, remoteAddr string, id string) {
	defer func() {
		conn.Close()
		outConn.Close()
	}()
	for {
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			log.Printf("%s [http ReadRequest from conn] %s - %s : %s", id, conn.RemoteAddr(), conn.LocalAddr(), err)
			return
		}

		// log.Printf("%s [http] %s - %s : %v -> ", id, conn.RemoteAddr(), conn.LocalAddr(), req.URL)

		resp, err := handleRequest(conn, outConn, req, remoteAddr)
		if err != nil {
			log.Printf("%s [http handleRequest] %s - %s : %s", id, conn.RemoteAddr(), conn.LocalAddr(), err)
			return
		}

		// log.Printf("%s [http] %s - %s : %v -> %s <-", id, conn.RemoteAddr(), conn.LocalAddr(), req.URL, resp.Status)

		err = resp.Write(conn)
		if err != nil {
			log.Printf("%s [http Write to conn] %s - %s : %s", id, conn.RemoteAddr(), conn.LocalAddr(), err)
			return
		}

		// log.Printf("%s [http finished] %s - %s : %v -> %s <-", id, conn.RemoteAddr(), conn.LocalAddr(), req.URL, resp.Status)

		return
	}
}
