package main

import (
	"ZhatRoom/internal/config"
	"ZhatRoom/internal/server"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfgPath := flag.String("config", "/opt/zhatroom/config.yaml", "config file path")
	flag.Parse()

	var cfg config.ServerConfig
	if err := config.Load(*cfgPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	srv := server.NewServer(cfg)
	if err := srv.Listen(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
		os.Exit(1)
	}
	srv.AcceptLoop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	srv.Shutdown()
}
