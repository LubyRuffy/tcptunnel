package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"net/http"

	"github.com/xtaci/smux"
)

func connectOneServer(publicServerAddr string, registerID string, remoteServerAddr string) (err error) {
	cli, err := net.DialTimeout("tcp", publicServerAddr, time.Second*10)
	if err != nil {
		return errors.New(fmt.Sprintf("Dial to server %s failed: %v!", publicServerAddr, err))
	}

	log.Println("connected to public server : ", publicServerAddr)

	defer func() {
		cli.Close()
	}()

	session, _ := smux.Client(cli, nil)
	stream, _ := session.OpenStream()

	defer func() {
		stream.Close()
		session.Close()
	}()

	buf := make([]byte, 65536)
	var n int

	stream.Write([]byte(fmt.Sprintf("REGISTER /%s HTTP/1.0\r\n\r\n", registerID)))
	n, err = stream.Read(buf)
	if err != nil {
		return errors.New(fmt.Sprintf("stream.Read failed: %v", err))
	}

	var req *http.Request
	for {
		req, err = recvReq(stream)
		if err != nil {
			return errors.New(fmt.Sprintf("recvReq failed: %v", err))
		}

		switch req.Method {
		case "NEWDATASTREAM":
			stream.Write([]byte("200 OK\r\n\r\n"))
			go func(id string) {
				stream, _ := session.OpenStream()
				defer func() {
					stream.Close()
				}()
				n, err = stream.Write([]byte(fmt.Sprintf("DATASTREAM %s HTTP/1.0\r\n\r\n", id)))
				if err != nil {
					//errors.New(fmt.Sprintf("stream.Write failed: %v", err))
					return
				}
				n, err = stream.Read(buf)
				if err != nil {
					log.Printf("stream.Read failed: %v\n", err)
					return
				}
				log.Printf("stream.Read : %s\n", string(buf[:n]))
				log.Printf("try to net.Dial to %s\n", remoteServerAddr)

				outConn, err := net.Dial("tcp", remoteServerAddr)
				if err != nil {
					log.Printf("net.Dial failed: %v", err)
					return
				}

				log.Printf("net.Dial to %s ok\n", remoteServerAddr)

				IoBind(stream, outConn, func(err interface{}) {
					if err != io.EOF && err != nil {
						log.Printf("IoBind failed: %v\n", err)
					}

					inAddr := stream.RemoteAddr().String()
					outAddr := outConn.RemoteAddr().String()
					log.Printf("conn %s - %s released", inAddr, outAddr)
				})
			}(req.RequestURI)
		default:
			log.Println("Unknown msg type")
		}
	}

	return
}

// 控制stream通道
func connectServer(wg *sync.WaitGroup, publicServerAddr string, registerID string, remoteServerAddr string) {
	defer func() {
		wg.Done()
	}()

	var err error
	for {
		err = connectOneServer(publicServerAddr, registerID, remoteServerAddr)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}
}

func natServer() {
	wg := sync.WaitGroup{}
	for _, v := range configOptions.NatServer {
		fmt.Printf("ID %s -> %s \n", v.ID, v.RemoteServerAddr)
		go connectServer(&wg, configOptions.PublicServerAddr, v.ID, v.RemoteServerAddr)
		wg.Add(1)
	}
	wg.Wait()
}
