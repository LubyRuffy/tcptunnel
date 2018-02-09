package main

import (
	"log"
)

const Version = "1.0"

func PrintVersion() {
	log.Println("====================")
	log.Println("tcptunnel, version ", Version)
	log.Println("https://github.com/LubyRuffy/tcptunnel")
	log.Println("====================")
}
