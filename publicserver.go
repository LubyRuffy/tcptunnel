package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/xtaci/smux"
)

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

func publicServer() {
	rand.Seed(time.Now().UnixNano())
	wg := sync.WaitGroup{}
	listenPublicServer(&wg, configOptions.PublicServer.LocalBindAddr, func(stream *smux.Stream) {

		buf := make([]byte, 1024*1024)
		req, err := recvReq(stream)
		if err != nil {
			log.Printf("recvReq failed: %v", err)
			return
		}

		// log.Println("recv req: ", req)
		switch req.Method {
		case "CONNECT":
			log.Println("CONNECT msg to ", req.RequestURI)
			if ctlStream, ok := natCtlMap.Load(req.RequestURI); ok {
				dataStreamID := "/" + random(1000, 10000000)
				// log.Println(" ctlStream: ", ctlStream, " dataStreamID: ", dataStreamID)

				// 新建等待事件
				dataStreamMap.Store(dataStreamID, make(chan *smux.Stream))
				// log.Println("wait DATASTREAM event:", dataStreamID)

				// 通知建立新的数据通道
				_, err = ctlStream.(*smux.Stream).Write([]byte(fmt.Sprintf("NEWDATASTREAM %s HTTP/1.0\r\n\r\n", dataStreamID)))
				if err != nil {
					log.Println(" stream write data error: ", err)
				}

				_, err := ctlStream.(*smux.Stream).Read(buf)
				if err != nil {
					log.Println(" recvReq error: ", err)
				}
				// log.Println("recv data req: ", string(buf[:n]))

				// 等待通知
				waitChan, ok := dataStreamMap.Load(dataStreamID)
				if !ok {
					panic(fmt.Sprintf(" dataStreamMap not found of : %s", dataStreamID))
				}
				dataStream := <-waitChan.(chan *smux.Stream)
				// log.Println("DATASTREAM event ok: ", dataStream)

				stream.Write([]byte("200 OK\r\n\r\n"))

				IoBind(dataStream, stream, func(err interface{}) {
					if err != io.EOF && err != nil {
						log.Printf("IoBind failed: %v\n", err)
					}
				})
			} else {
				log.Println("Could not found stream of id : ", req.RequestURI)
			}
		case "REGISTER":
			log.Println("REGISTER msg to ", req.RequestURI)
			natCtlMap.Store(req.RequestURI, stream)
			stream.Write([]byte("200 OK\r\n\r\n"))
			// log.Println("DUMP====")
			// natCtlMap.Range(func(key, value interface{}) bool {
			// 	log.Println(key, value)
			// 	return true
			// })
			// log.Println("====DUMP")
		case "DATASTREAM":
			log.Println("DATASTREAM msg to ", req.RequestURI)
			stream.Write([]byte("200 OK\r\n\r\n"))
			waitChan, ok := dataStreamMap.Load(req.RequestURI)
			if !ok {
				log.Println(" dataStreamMap not found of : ", req.RequestURI)
			}
			waitChan.(chan *smux.Stream) <- stream
		default:
			log.Println("Unknown msg type")
			return
		}
	})
}
