package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/xtaci/smux"
)

type ControlSession struct {
	Session *smux.Session
}

// 控制stream通道
func getControlSession(addr string) (session *smux.Session, err error) {
	cli, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	session, err = smux.Client(cli, nil)
	if err != nil {
		return
	}
	return
}

func bindConnToServer(id string, newconn net.Conn, session *ControlSession) {
	var stream *smux.Stream
	var err error

	defer func() {
		newconn.Close()
		if stream != nil {
			stream.Close()
		}
	}()

	if session.Session == nil {
		log.Println("[ERROR] session.Session not connected!")
		return
	}

	buf := make([]byte, 1024*1024)
	stream, err = session.Session.OpenStream()

	if err != nil {
		log.Println("session.OpenStream error:", err)
		return
	}
	log.Println("session.OpenStream ok, id is :", stream.ID())

	n, err := stream.Write([]byte(fmt.Sprintf("CONNECT /%s HTTP/1.0\r\n\r\n", id)))
	if err != nil {
		log.Println("stream.Write CONNECT error:", err)
		return
	}

	// 10秒读取超时
	stream.SetReadDeadline(time.Now().Add(time.Duration(10) * time.Second))
	n, err = stream.Read(buf)
	if err != nil {
		log.Println("stream.Read error:", err)
		return
	}

	log.Println("stream.Read size : ", n)
	if strings.Contains(string(buf[:n]), "200 OK") {
		log.Printf("CONNECT to server OK\n")

		stream.SetReadDeadline(time.Time{})
		IoBind(newconn, stream, func(err interface{}) {
			if err != io.EOF && err != nil {
				log.Printf("IoBind failed: %v\n", err)
			}
		})
	} else {
		log.Printf("CONNECT to server failed: %s\n", string(buf[:n]))
		return
	}
}

func clientConnect() {
	session := ControlSession{}
	var err error

	go func(ctlsession *ControlSession) {
		// 获取控制session
		for {
			if ctlsession.Session != nil && !ctlsession.Session.IsClosed() {
				time.Sleep(time.Second * 5)
			} else {
				ctlsession.Session, err = getControlSession(configOptions.PublicServerAddr)
				if err != nil {
					log.Printf("connect to server failed: %v, retry after 5 seconds\n", err)
					time.Sleep(time.Second * 5)
					continue
				}
			}
		}
	}(&session)

	wg := sync.WaitGroup{}

	// 监听
	for _, v := range configOptions.ClientConnect {

		go func(config ClientConnectConfig) {
			listenTCPServer(&wg, config.LocalBindAddr, func(newconn net.Conn) {
				bindConnToServer(config.ID, newconn, &session)
			})
		}(v)
		wg.Add(1)
	}

	wg.Wait()
}
