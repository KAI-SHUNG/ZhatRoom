package main

import (
	"ZhatRoom/internal/config"
	"ZhatRoom/internal/protocol"
	"bufio"
	"fmt"
	"net"
)

func main() {
	var serverConfig config.ServerConfig
	if err := config.LoadConfig("./config.yaml", &serverConfig); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	router := protocol.NewRouter()

	ln, err := net.Listen("tcp", serverConfig.Config.Addr)
	if err != nil {
		fmt.Printf("listen error: %v\n", err)
		return
	}
	fmt.Printf("Server is listening on %s\n", serverConfig.Config.Addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("accept error: %v\n", err)
			continue
		}
		fmt.Printf("New connection from %s\n", conn.RemoteAddr())
		go handleConnection(conn, router)
	}
}

func handleConnection(conn net.Conn, router *protocol.Router) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("read error: %v\n", err)
			return
		}
		msg, err := protocol.FromJSON(line)
		if err != nil {
			fmt.Printf("invalid message format: %v\n", err)
			continue
		}
		fmt.Printf("Received message: %+v\n", msg)

		router.Dispatch(msg)
	}
}
