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

	msgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	sysStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("43"))
)

func renderMessages(msgs []protocol.Message) string {
	var b strings.Builder
	for _, msg := range msgs {
		if msg.Type == "system" {
			b.WriteString(sysStyle.Render(fmt.Sprintf("  %s\n", msg.Content)))
		} else {
			b.WriteString(msgStyle.Render(fmt.Sprintf("[%s]: %s\n", msg.From, msg.Content)))
		}
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
