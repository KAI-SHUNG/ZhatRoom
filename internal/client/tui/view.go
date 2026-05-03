package tui

import (
	"fmt"
	"strings"

	"ZhatRoom/internal/protocol"

	"github.com/charmbracelet/lipgloss"
)

var (
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			SetString(strings.Repeat("─", 30))

	msgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	sysStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("43"))
	otherStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

func renderMessages(msgs []protocol.Message, width int, myID string) string {
	var b strings.Builder
	for _, msg := range msgs {
		var line string
		if msg.Type == "system" {
			line = sysStyle.
				Width(width).
				Align(lipgloss.Center).
				Render(fmt.Sprintf("[SYSTEM]: %s", msg.Content))
		} else if msg.FromID == myID {
			line = msgStyle.
				Width(width).
				Align(lipgloss.Right).
				Render(fmt.Sprintf("[You]: %s", msg.Content))
		} else {
			line = otherStyle.
				Width(width).
				Align(lipgloss.Left).
				Render(fmt.Sprintf("[%s]: %s", msg.From, msg.Content))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Connecting to ZhatRoom..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Disconnected: %v\n", m.err)
	}

	view := m.viewport.View()
	view += "\n" + dividerStyle.String() + "\n"
	view += m.input.View()
	return view
}
