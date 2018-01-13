package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

func tcpProxyPair(newconn net.Conn, remoteAddr string) {
	defer func() {
		newconn.Close()
	}()

	outConn, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		panic(err)
		return
	}

	defer func() {
		outConn.Close()
	}()

	IoBind(newconn, outConn, func(err interface{}) {
		if err != io.EOF && err != nil {
			log.Printf("IoBind failed: %v\n", err)
		}

		inAddr := newconn.RemoteAddr().String()
		outAddr := outConn.RemoteAddr().String()
		log.Printf("newconn %s - %s released", inAddr, outAddr)
	})
}

// 端口转发
func createOneTcpProxy(wg *sync.WaitGroup, localBindAddr string, remoteAddr string) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("bind crashed : %s", err)
		}
		wg.Done()
	}()

	l, err := net.Listen("tcp", localBindAddr)
	if err == nil {
		log.Printf("bind server to %s ok, listenning...\n", localBindAddr)
		for {
			var conn net.Conn
			conn, err = l.Accept()
			if err == nil {
				go tcpProxyPair(conn, remoteAddr)
			} else {
				panic(err)
				return
			}
		}
	} else {
		panic("bind error")
	}
}

func tcpProxy() {
	wg := sync.WaitGroup{}
	for _, v := range configOptions.TcpProxies {
		fmt.Printf("proxy of %s -> %s \n", v.LocalBindAddr, v.RemoteServerAddr)
		go createOneTcpProxy(&wg, v.LocalBindAddr, v.RemoteServerAddr)
		wg.Add(1)
	}
	wg.Wait()
}
