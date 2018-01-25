package main

import (
	"flag"
	"fmt"
	// _ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/BurntSushi/toml"
	//"go/types"
)

type TcpProxyConfig struct {
	LocalBindAddr    string
	RemoteServerAddr string
	Type             string
}

type PublicServerConfig struct {
	LocalBindAddr string
}

type ClientConnectConfig struct {
	LocalBindAddr string
	ID            string
}

type NatServerConfig struct {
	RemoteServerAddr string
	ID               string
	Type             string
}

type TomlConfig struct {
	Mode             string
	PublicServerAddr string
	TcpProxies       map[string]TcpProxyConfig
	PublicServer     PublicServerConfig
	NatServer        map[string]NatServerConfig
	ClientConnect    map[string]ClientConnectConfig
}

var configOptions TomlConfig

func init() {
	cpuProfile := flag.String("p", "", "write cpu profile to file")
	configFile := flag.String("c", "config.toml", "config file")
	mode := flag.String("m", "", "mode, can overwrite config_file's setting")
	flag.Parse()

	if *configFile == "" {
		panic("config file is not specified")
	}

	if _, err := toml.DecodeFile(*configFile, &configOptions); err != nil {
		// handle error
		panic(err)
	}

	if *mode != "" {
		configOptions.Mode = *mode
	}

	if *cpuProfile != "" {
		// go func() {
		// 	log.Println(http.ListenAndServe("localhost:6060", nil))
		// }()

		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Println(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
}
