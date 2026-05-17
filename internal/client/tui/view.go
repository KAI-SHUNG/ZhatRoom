package tui

import (
	"fmt"
	"strings"
	"time"

	"ZhatRoom/internal/protocol"

	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 20

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
	cmdNameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cmdDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("249"))
	cmdHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("249"))
	cmdSelectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Background(lipgloss.Color("237")).Bold(true)
)

var (
	statusBarStyle      = lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("252")).Bold(true)
	sidebarStyle        = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252"))
	sidebarItemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sidebarActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("43")).Bold(true)
	sidebarCursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Background(lipgloss.Color("237")).Bold(true)
	modeIndicatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
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
	if m.mode == InputMode {
		hints := m.matchingCommands()
		return 3 + len(hints) // divider + input + hint lines
	}
	return 2 // divider + mode hint
}

func (m *Model) renderStatusBar() string {
	online := 0
	for _, r := range m.roomList {
		if r.ID == m.currentRoomID {
			online = r.Members
			break
		}
	}
	left := fmt.Sprintf(" #%s  %d online", m.currentRoomName, online)
	right := fmt.Sprintf(" %s ", m.modeString())

	gap := m.winWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return statusBarStyle.Width(m.winWidth).Render(left + strings.Repeat(" ", gap) + right)
}

func (m *Model) renderSidebar() string {
	var lines []string
	lines = append(lines, sidebarStyle.Width(sidebarWidth).Render("  ROOMS"))
	lines = append(lines, sidebarStyle.Width(sidebarWidth).Render("  "+strings.Repeat("─", sidebarWidth-4)))

	for i, r := range m.roomList {
		prefix := "  "
		style := sidebarItemStyle
		if r.ID == m.currentRoomID {
			prefix = "▸ "
			style = sidebarActiveStyle
		}
		if i == m.sidebarCursor && m.mode == SidebarMode {
			style = sidebarCursorStyle
			prefix = "▸ "
		}
		name := r.Name
		if len(name) > sidebarWidth-8 {
			name = name[:sidebarWidth-11] + "..."
		}
		line := fmt.Sprintf("%s%s (%d)", prefix, name, r.Members)
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(style.Render(line)))
	}

	// Fill remaining height
	remaining := m.winHeight - 2 - len(lines) // -2 for status bar and divider
	for i := 0; i < remaining; i++ {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Connecting to ZhatRoom..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Disconnected: %v\n", m.err)
	}

	// Status bar (always visible)
	statusBar := m.renderStatusBar()

	// Main content area
	var mainContent string
	chatWidth := m.winWidth
	if m.mode != InputMode {
		sidebar := m.renderSidebar()
		chatWidth = m.winWidth - sidebarWidth
		m.viewport.Width = chatWidth
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, m.viewport.View())
	} else {
		m.viewport.Width = chatWidth
		mainContent = m.viewport.View()
	}

	// Divider
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		SetString(strings.Repeat("─", max(chatWidth, 1))).String()

	// Footer
	var footer string
	if m.mode == InputMode {
		footer = m.input.View()
		hints := m.matchingCommands()
		if len(hints) > 0 {
			for i, h := range hints {
				selected := i == m.cmdIdx
				if selected {
					name := cmdSelectStyle.Render("/" + h.name)
					desc := cmdSelectStyle.Render(" " + h.desc)
					footer += "\n" + "▸ " + name + desc
				} else {
					name := cmdNameStyle.Render("/" + h.name)
					desc := cmdDescStyle.Render(h.desc)
					footer += "\n" + cmdHintStyle.Render("  ") + name + "  " + desc
				}
			}
		}
	} else {
		var modeHint string
		switch m.mode {
		case NormalMode:
			modeHint = "  [i] input  [h] sidebar  [q] quit"
		case SidebarMode:
			modeHint = "  [j/k] navigate  [Enter] switch  [l] back"
		}
		footer = modeIndicatorStyle.Render(modeHint)
	}

	return statusBar + "\n" + mainContent + "\n" + divider + "\n" + footer
}
