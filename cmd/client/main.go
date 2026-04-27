package main

import (
	"ZhatRoom/internal/client/tui"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	id := flag.String("id", "001", "ZhatRoom Id")
	nickname := flag.String("usr", "Anonymous", "ZhatRoom nickname")
	flag.Parse()

	connector, err := tui.NewConnector("/tmp/zhatroom.sock", *id, *nickname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer connector.Close()

	p := tea.NewProgram(tui.NewModel(*id, *nickname, connector), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
