package main

import (
	"ZhatRoom/internal/client/tui"
	"ZhatRoom/internal/config"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfgPath := flag.String("config", "/opt/zhatroom/config.yaml", "config file path")
	id := flag.String("id", "001", "ZhatRoom Id")
	nickname := flag.String("usr", "", "ZhatRoom nickname")
	socket := flag.String("socket", "", "Unix socket path")
	flag.Parse()

	var cfg config.ClientConfig
	if err := config.Load(*cfgPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// CLI flags override config
	if *socket != "" {
		cfg.Socket = *socket
	}
	if *nickname != "" {
		cfg.Username = *nickname
	}

	connector, err := tui.NewConnector(cfg.Socket, *id, cfg.Username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer connector.Close()

	p := tea.NewProgram(tui.NewModel(*id, cfg.Username, connector), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
