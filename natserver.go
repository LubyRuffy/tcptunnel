package main

import (
	"bufio"
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

func doNewDataStream(id string, stream *smux.Stream, v NatServerConfig) {
	defer func() {
		log.Printf("close stream id: %v\n", id)
		stream.Close()
	}()

	buf := make([]byte, 65536)

	_, err := stream.Write([]byte(fmt.Sprintf("DATASTREAM %s HTTP/1.0\r\n\r\n", id)))
	if err != nil {
		log.Println(fmt.Sprintf("stream.Write failed: %v", err))
		return
	}

	_, err = stream.Read(buf)
	if err != nil {
		log.Printf("stream.Read failed: %v\n", err)
		return
	}
	//log.Printf("%s stream.Read : %s\n", id, string(buf[:n]))
	// log.Printf("%s try to net.Dial to %s\n", id, v.RemoteServerAddr)

	outConn, err := net.Dial("tcp", v.RemoteServerAddr)
	if err != nil {
		log.Printf("%s net.Dial failed: %v", id, err)
		return
	}
	// log.Printf("%s net.Dial to %s ok\n", id, v.RemoteServerAddr)

	if v.Type == "http" {
		HTTPBind(stream, outConn, v.RemoteServerAddr, id)
	} else {
		IoBind(stream, outConn, func(err interface{}) {
			if err != io.EOF && err != nil {
				log.Printf("IoBind failed: %v\n", err)
			}

			// inAddr := stream.RemoteAddr().String()
			// outAddr := outConn.RemoteAddr().String()
			// log.Printf("conn %s - %s released", inAddr, outAddr)
		})
	}

}

func connectOneServer(publicServerAddr string, v NatServerConfig) (err error) {
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

	stream.Write([]byte(fmt.Sprintf("REGISTER /%s HTTP/1.0\r\n\r\n", v.ID)))
	n, err := stream.Read(buf)
	if err != nil {
		return errors.New(fmt.Sprintf("stream.Read failed: %v", err))
	}
	log.Println(string(buf[:n]))

	var req *http.Request
	for {
		req, err = http.ReadRequest(bufio.NewReader(stream))
		if err != nil {
			return errors.New(fmt.Sprintf("recvReq failed: %v", err))
		}

		switch req.Method {
		case "NEWDATASTREAM":
			// stream.Write([]byte("200 OK\r\n\r\n"))
			log.Println("NEWDATASTREAM", req.RequestURI)
			newstream, _ := session.OpenStream()
			go doNewDataStream(req.RequestURI, newstream, v)
		default:
			log.Println("Unknown msg type")
		}
	}

	return
}

// 控制stream通道
func connectServer(wg *sync.WaitGroup, publicServerAddr string, v NatServerConfig) {
	defer func() {
		wg.Done()
	}()

	var err error
	for {
		err = connectOneServer(publicServerAddr, v)
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
		go connectServer(&wg, configOptions.PublicServerAddr, v)
		wg.Add(1)
	}
	wg.Wait()
}
