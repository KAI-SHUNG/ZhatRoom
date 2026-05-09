package main

import (
	"ZhatRoom/internal/config"
	"ZhatRoom/internal/server"
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	var cfg config.ServerConfig
	if err := config.Load(*cfgPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	exitChan := make(chan bool, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\nReceived shutdown signal, exiting...")
		exitChan <- true
	}()

	if err := os.Remove(cfg.Socket); err != nil && !os.IsNotExist(err) {
		fmt.Printf("[Server]: remove error: %v\n", err)
		return
	}
	fmt.Print("[Server]: Removed existing socket file.\n")
	ln, err := net.Listen("unix", cfg.Socket)
	if err != nil {
		fmt.Printf("[Server]: listen error: %v\n", err)
		return
	}
	fmt.Printf("[Server]: Server is listening on %s\n", cfg.Socket)

	hub := server.NewHub(cfg)
	go hub.Run()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					fmt.Println("[Server]: Server is shutting down, stop accepting new connections.")
					return
				}
				fmt.Printf("[Server]: accept error: %v\n", err)
				continue
			}
			fmt.Printf("[Server]: Connection established from %s\n", conn.RemoteAddr())

			reader := bufio.NewReader(conn)
			line, err := reader.ReadBytes('\n')
			if err != nil {
				fmt.Printf("[Server]: read error: %v\n", err)
				conn.Close()
				continue
			}
			s := strings.TrimSpace(string(line))

			if strings.HasPrefix(s, "validate,") {
				uid := strings.TrimPrefix(s, "validate,")
				if hub.Validate(uid) {
					fmt.Fprint(conn, "ok\n")
				} else {
					fmt.Fprint(conn, "denied\n")
				}
				conn.Close()
				continue
			}

			parts := strings.Split(s, ",")
			if len(parts) != 2 {
				fmt.Println("[Server]: Invalid client info format")
				conn.Close()
				continue
			}
			id, nickname := parts[0], parts[1]

			fmt.Printf("[Server]: Handling connection from id: %s, nickname: %s\n", id, nickname)
			hub.HandleNewConn(conn, id, nickname)
		}
	}()

	<-exitChan
	fmt.Println("[Server]: Shutting down server...")
	ln.Close()
	wg.Wait()
	hub.Shutdown()
}
