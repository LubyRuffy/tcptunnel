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
	"time"

	"github.com/xtaci/smux"
)

type ControlStream struct {
	sync.Mutex
	Stream *smux.Stream
}

var natCtlMap sync.Map     // 控制流
var dataStreamMap sync.Map // 数据通道流

// 公网监听服务器
func listenPublicServer(wg *sync.WaitGroup, addr string, fn onStream) (err error) {
	err = listenTCPServer(wg, addr, func(newconn net.Conn) {
		sess, err := smux.Server(newconn, nil)
		if err != nil {
			log.Printf("smux should not fail with default config: %v", err)
			return
		}

		for {
			stream, err := sess.AcceptStream()
			if err != nil {
				log.Printf("AcceptStream failed: %v", err)
				return
			}
			// log.Printf("AcceptStream ok, id: %v", stream.ID)

			go func(newstream *smux.Stream) {
				fn(newstream)
			}(stream)
		}
	})
	return
}

func doConnect(req *http.Request, stream *smux.Stream) {
	var err error
	// buf := make([]byte, 1024*1024)

	if ctlStream, ok := natCtlMap.Load(req.RequestURI); ok {
		dataStreamID := "/" + random(1000, 10000000)
		// log.Println(" ctlStream: ", ctlStream, " dataStreamID: ", dataStreamID)

		// 新建等待事件
		dataStreamMap.Store(dataStreamID, make(chan *smux.Stream, 1))
		// log.Println("wait DATASTREAM event:", dataStreamID)
		defer func() {
			dataStreamMap.Delete(dataStreamID)
		}()

		// 通知建立新的数据通道
		log.Println("send NEWDATASTREAM", dataStreamID)
		ctlStream.(*ControlStream).Lock()
		_, err = ctlStream.(*ControlStream).Stream.Write([]byte(fmt.Sprintf("NEWDATASTREAM %s HTTP/1.0\r\n\r\n", dataStreamID)))
		time.Sleep(time.Millisecond * 20) // 这里必须延时一下，否则在浏览器并发连接的时候，natserver在readRequest的过程中会接受不到数据，这个需要再想办法，目前临时加一个延时
		ctlStream.(*ControlStream).Unlock()
		if err != nil {
			log.Println(" stream write data error: ", err)
		}

		// 等待通知
		waitChan, ok := dataStreamMap.Load(dataStreamID)
		if !ok {
			panic(fmt.Sprintf(" dataStreamMap not found of : %s", dataStreamID))
		}

		// dataStream := <-waitChan.(chan *smux.Stream)
		// log.Println("DATASTREAM event ok: ", dataStream)
		var dataStream *smux.Stream
		select {
		case dataStream = <-waitChan.(chan *smux.Stream):
			// log.Println("DATASTREAM event ok: ", dataStream)
			dataStreamMap.Delete(dataStreamID)
		case <-time.After(time.Second * 10):
			log.Println("waitChan DATASTREAM timeout: ", dataStreamID)
			dataStreamMap.Delete(dataStreamID)
			return
		}

		// log.Println("DUMP 222222")
		// dataStreamMap.Range(func(key, value interface{}) bool {
		// 	log.Println(key, value)
		// 	return true
		// })
		// log.Println("222222 DUMP")

		stream.Write([]byte("200 OK\r\n\r\n"))

		IoBind(dataStream, stream, func(err interface{}) {
			if err != io.EOF && err != nil {
				log.Printf("IoBind failed: %v\n", err)
			}
		})
	} else {
		log.Println("Could not found stream of id : ", req.RequestURI)
	}
}

func doDataStream(req *http.Request, stream *smux.Stream) {
	log.Println("DATASTREAM msg to ", req.RequestURI)
	stream.Write([]byte("200 OK\r\n\r\n"))
	waitChan, ok := dataStreamMap.Load(req.RequestURI)
	if ok {
		waitChan.(chan *smux.Stream) <- stream
	} else {
		log.Println(" dataStreamMap not found of : ", req.RequestURI)
	}
}

func publicServer() {
	rand.Seed(time.Now().UnixNano())
	wg := sync.WaitGroup{}

	listenPublicServer(&wg, configOptions.PublicServer.LocalBindAddr, func(stream *smux.Stream) {
		req, err := http.ReadRequest(bufio.NewReader(stream))
		if err != nil {
			log.Printf("recv request failed: %v", err)
			return
		}

		// log.Println("recv req: ", req)
		switch req.Method {
		case "CONNECT":
			// log.Println("CONNECT msg to ", req.RequestURI)
			go doConnect(req, stream)
		case "REGISTER":
			log.Println("REGISTER msg to ", req.RequestURI)
			ctlStream := ControlStream{Stream: stream}
			natCtlMap.Store(req.RequestURI, &ctlStream)
			stream.Write([]byte("200 OK\r\n\r\n"))
			// log.Println("DUMP====")
			// natCtlMap.Range(func(key, value interface{}) bool {
			// 	log.Println(key, value)
			// 	return true
			// })
			// log.Println("====DUMP")
		case "DATASTREAM":
			go doDataStream(req, stream)

		default:
			log.Println("Unknown msg type")
			return
		}
	})
}
