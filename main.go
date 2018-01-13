package main

import (
	"fmt"
	"log"
)

func main() {
	log.Println("Mode is :", configOptions.Mode)
	switch configOptions.Mode {
	case "tcpproxy":
		tcpProxy()
	case "natserver":
		natServer()
	case "publicserver":
		publicServer()
	case "client":
		clientConnect()
	default:
		fmt.Println("unknow mode of:", configOptions.Mode)
	}
}
