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

	msgStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	sysStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("43"))
	otherStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("237"))
	inputHLStyle   = lipgloss.NewStyle().Background(lipgloss.Color("236"))
)

var timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

var (
	cmdNameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cmdDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("249"))
	cmdHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("249"))
	cmdSelectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Background(lipgloss.Color("237")).Bold(true)
)

var (
	statusBarStyle     = lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("252")).Bold(true)
	sidebarHeaderStyle = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("249")).Bold(true)
	sidebarStyle       = lipgloss.NewStyle().Background(lipgloss.Color("234")).Foreground(lipgloss.Color("243"))
	sidebarItemStyle   = lipgloss.NewStyle().Background(lipgloss.Color("234")).Foreground(lipgloss.Color("243"))
	sidebarActiveStyle = lipgloss.NewStyle().Background(lipgloss.Color("234")).Foreground(lipgloss.Color("252")).Bold(true)
	sidebarCursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("239")).Foreground(lipgloss.Color("15")).Bold(true)
	chatBgStyle        = lipgloss.NewStyle().Background(lipgloss.Color("235"))
)

var (
	modeInputStyle = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252"))
	modeEscStyle   = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252"))
)

func renderMessages(msgs []protocol.Message, width int, myID string, cursorLine int) (string, int, []int) {
	var b strings.Builder
	var prevTS int64
	visualLine := 0
	var lineMap []int

	for i, msg := range msgs {
		if msg.Type == "history_end" {
			continue
		}

		// insert centered timestamp when gap > 5 minutes
		if msg.CreatedAt > 0 && prevTS > 0 && msg.CreatedAt-prevTS > 300 {
			ts := time.Unix(msg.CreatedAt, 0).Format("15:04")
			label := "─── " + ts + " ───"
			b.WriteString(timestampStyle.Width(width).Align(lipgloss.Center).Render(label) + "\n")
			lineMap = append(lineMap, -1)
			visualLine++
		}
		if msg.CreatedAt > 0 {
			prevTS = msg.CreatedAt
		}

		// blank line before non-system messages (except at the start)
		if msg.Type != "system" && b.Len() > 0 {
			b.WriteString("\n")
			lineMap = append(lineMap, -1)
			visualLine++
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
		if visualLine == cursorLine {
			line = highlightStyle.Render(line)
		}
		b.WriteString(line + "\n")

		// Calculate visual lines for this message
		content := msg.Content
		if msg.Type == "system" {
			content = fmt.Sprintf("[SYSTEM]: %s", msg.Content)
		} else if msg.FromID == myID {
			content = fmt.Sprintf("[You]: %s", msg.Content)
		} else {
			content = fmt.Sprintf("[%s]: %s", msg.From, msg.Content)
		}
		vLines := 1
		for _, part := range strings.Split(content, "\n") {
			if width > 0 && len(part) > width {
				vLines += (len(part) - 1) / width
			}
		}
		for j := 0; j < vLines; j++ {
			lineMap = append(lineMap, i)
			visualLine++
		}
	}
	return b.String(), visualLine, lineMap
}

func (m *Model) matchingCommands() []cmdEntry {
	val := m.input.Value()
	if len(val) == 0 || val[0] != '/' {
		return nil
	}

	// Check for /room subcommands
	lower := strings.ToLower(val)
	if strings.HasPrefix(lower, "/room ") {
		subPrefix := strings.TrimPrefix(lower, "/room ")
		var matches []cmdEntry
		for _, c := range roomSubCommands {
			if strings.HasPrefix(c.name, subPrefix) {
				matches = append(matches, c)
			}
		}
		return matches
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
	// divider(1) + input(1) + hints + mode indicator(1)
	return 3 + len(hints)
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

	chatWidth := m.winWidth - sidebarWidth
	gap := chatWidth - lipgloss.Width(left)
	if gap < 0 {
		gap = 0
	}
	sbPad := sidebarStyle.Width(sidebarWidth).Render("")
	return sbPad + statusBarStyle.Width(chatWidth).Render(left+strings.Repeat(" ", gap))
}

func (m *Model) renderSidebar() string {
	var lines []string
	// Header
	lines = append(lines, sidebarHeaderStyle.Width(sidebarWidth).Render("  ROOMS"))
	lines = append(lines, sidebarHeaderStyle.Width(sidebarWidth).Render("  "+strings.Repeat("─", sidebarWidth-4)))

	for i, r := range m.roomList {
		prefix := "  "
		style := sidebarItemStyle
		if r.ID == m.currentRoomID {
			prefix = "▸ "
			style = sidebarActiveStyle
		}
		if m.mode == SidebarMode && i == m.sidebarCursor {
			style = sidebarCursorStyle
			prefix = "▸ "
		}
		name := r.Name
		if len(name) > sidebarWidth-8 {
			name = name[:sidebarWidth-11] + "..."
		}
		line := fmt.Sprintf("%s%s (%d)", prefix, name, r.Members)
		lines = append(lines, style.Width(sidebarWidth).Render(line))
	}

	// Fill remaining height to match viewport + footer
	footerH := m.footerHeight()
	remaining := m.winHeight - 1 - footerH - len(lines) // -statusBar -footer
	for i := 0; i < remaining; i++ {
		lines = append(lines, sidebarStyle.Width(sidebarWidth).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderFooter() string {
	chatWidth := m.winWidth - sidebarWidth

	// Divider
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Repeat("─", max(chatWidth, 1)))

	// Input line
	inputLine := m.input.View()
	if m.mode == NormalMode {
		inputLine = inputHLStyle.Render("▸ " + strings.TrimLeft(inputLine, " "))
	}

	// Sidebar prefix for each line
	sbPad := sidebarStyle.Width(sidebarWidth).Render("")
	prefix := sbPad

	// Compose: divider + input (top-aligned) + hints + mode
	footer := prefix + divider + "\n" +
		prefix + " " + inputLine

	// Hint lines (each prefixed with sidebar + separator)
	hints := m.matchingCommands()
	if len(hints) > 0 {
		for i, h := range hints {
			var line string
			if i == m.cmdIdx {
				name := cmdSelectStyle.Render("/" + h.name)
				desc := cmdSelectStyle.Render(" " + h.desc)
				line = "▸ " + name + desc
			} else {
				name := cmdNameStyle.Render("/" + h.name)
				desc := cmdDescStyle.Render(h.desc)
				line = cmdHintStyle.Render("  ") + name + "  " + desc
			}
			footer += "\n" + prefix + line
		}
	}

	// Mode indicator
	var modeText string
	switch m.mode {
	case InputMode:
		modeText = modeInputStyle.Width(chatWidth).Render("-- INPUT --")
	case NormalMode:
		modeText = lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("252")).Width(chatWidth).Render("-- NORMAL --")
	default:
		modeText = modeEscStyle.Width(chatWidth).Render("-- ESC --")
	}
	footer += "\n" + prefix + modeText

	return footer
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Connecting to ZhatRoom..."
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Disconnected: %v\n", m.err)
	}

	// Status bar (always visible, full width)
	statusBar := m.renderStatusBar()

	// Sidebar + chat viewport
	sidebar := m.renderSidebar()
	chatWidth := m.winWidth - sidebarWidth
	m.viewport.Width = chatWidth
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, m.viewport.View())

	// Footer
	footer := m.renderFooter()

	return statusBar + "\n" + mainContent + "\n" + footer
}
