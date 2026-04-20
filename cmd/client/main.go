package main

import (
	"ZhatRoom/internal/config"
	"ZhatRoom/internal/protocol"
	"fmt"
	"net"
)

func main() {
	var clientConfig config.ClientConfig
	if err := config.LoadConfig("./config.yaml", &clientConfig); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	router := protocol.NewRouter()

	conn, err := net.Dial("tcp", clientConfig.Config.Addr)
	if err != nil {
		fmt.Printf("dial error: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Printf("Connected to server at %s\n", clientConfig.Config.Addr)

	for {

	}
}
