package tui

import (
	"fmt"
	"strings"
	"time"

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

var timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

var (
	cmdNameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	cmdDescStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cmdHintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cmdSelectStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Background(lipgloss.Color("237")).Bold(true)
)

func renderMessages(msgs []protocol.Message, width int, myID string) string {
	var b strings.Builder
	var prevTS int64

	for _, msg := range msgs {
		if msg.Type == "history_end" {
			continue
		}

		// insert centered timestamp when gap > 5 minutes
		if msg.CreatedAt > 0 && prevTS > 0 && msg.CreatedAt-prevTS > 300 {
			ts := time.Unix(msg.CreatedAt, 0).Format("15:04")
			label := "─── " + ts + " ───"
			b.WriteString(timestampStyle.Width(width).Align(lipgloss.Center).Render(label) + "\n")
		}
		if msg.CreatedAt > 0 {
			prevTS = msg.CreatedAt
		}

		// blank line before non-system messages (except at the start)
		if msg.Type != "system" && b.Len() > 0 {
			b.WriteString("\n")
		}

		var line string
		switch {
		case msg.Type == "system":
			line = sysStyle.
				Width(width).
				Align(lipgloss.Center).
				Render(fmt.Sprintf("[SYSTEM]: %s", msg.Content))
		case msg.FromID == myID:
			line = msgStyle.
				Width(width).
				Align(lipgloss.Right).
				Render(fmt.Sprintf("[You]: %s", msg.Content))
		default:
			line = otherStyle.
				Width(width).
				Align(lipgloss.Left).
				Render(fmt.Sprintf("[%s]: %s", msg.From, msg.Content))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m *Model) matchingCommands() []cmdEntry {
	val := m.input.Value()
	if len(val) == 0 || val[0] != '/' {
		return nil
	}
	prefix := strings.ToLower(val[1:])
	var matches []cmdEntry
	for _, c := range builtinCommands {
		if strings.HasPrefix(c.name, prefix) {
			matches = append(matches, c)
		}
	}
	return matches
}

func (m *Model) footerHeight() int {
	hints := m.matchingCommands()
	return 3 + len(hints) // divider + input + hint lines
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Connecting to ZhatRoom..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Disconnected: %v\n", m.err)
	}

	hints := m.matchingCommands()

	view := m.viewport.View()
	view += "\n" + dividerStyle.String() + "\n"
	view += m.input.View()

	if len(hints) > 0 {
		view += "\n"
		for i, h := range hints {
			selected := i == m.cmdIdx
			if selected {
				name := cmdSelectStyle.Render("/" + h.name)
				desc := cmdSelectStyle.Render(" " + h.desc)
				view += "▸ " + name + desc + "\n"
			} else {
				name := cmdNameStyle.Render("/" + h.name)
				desc := cmdDescStyle.Render(h.desc)
				view += cmdHintStyle.Render("  ") + name + "  " + desc + "\n"
			}
		}
		view = strings.TrimRight(view, "\n")
	}

	return view
}
