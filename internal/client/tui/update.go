package tui

import (
	"fmt"

	"ZhatRoom/internal/protocol"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		headerHeight := 0
		footerHeight := 3
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - footerHeight
		m.viewport.YPosition = headerHeight
		m.input.Width = msg.Width - 4
		if !m.welcomeSent && m.viewport.Height > 0 {
			m.messages = append(m.messages, protocol.Message{
				Type:    "system",
				From:    "System",
				Content: fmt.Sprintf("Welcome to ZhatRoom! You are %s (%s)", m.nickname, m.id),
			})
			m.viewport.SetContent(renderMessages(m.messages, m.viewport.Width, m.id))
			m.viewport.GotoBottom()
			m.welcomeSent = true
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEnter:
			text := m.input.Value()
			m.input.SetValue("")

			if text == "" {
				return m, nil
			}

			if text == "/exit" {
				return m, tea.Quit
			}

			pMsg := &protocol.Message{
				Type:    "chat",
				From:    m.nickname,
				FromID:  m.id,
				Content: text,
			}
			if text[0] == '/' {
				pMsg.Type = "command"
			}

			if err := m.connector.Send(pMsg); err != nil {
				m.err = err
				return m, tea.Quit
			}
			return m, nil

		case tea.KeyPgUp:
			m.viewport.HalfPageUp()
			m.tryLoadHistory()
			return m, nil

		case tea.KeyPgDown:
			m.viewport.HalfPageDown()
			return m, nil
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case incomingMsg:
		m.handleIncoming(msg.msg)
		cmds = append(cmds, waitForMessage(m.connector))

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		if msg.Button == tea.MouseButtonWheelUp {
			m.tryLoadHistory()
		}
		cmds = append(cmds, cmd)

	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleIncoming(msg *protocol.Message) {
	switch msg.Type {
	case "history":
		m.pendingHistory = append(m.pendingHistory, *msg)
		return

	case "history_end":
		m.flushHistory()
		m.historyLoading = false
		if len(m.pendingHistory) == 0 && !m.historyEnd {
			m.historyEnd = true
		}
		return

	default:
		// flush any pending history before appending new message
		if len(m.pendingHistory) > 0 {
			m.flushHistory()
		}
		m.messages = append(m.messages, *msg)
		if msg.CreatedAt > 0 && (msg.CreatedAt < m.oldestTS || m.oldestTS == 0) {
			m.oldestTS = msg.CreatedAt
		}
		m.viewport.SetContent(renderMessages(m.messages, m.viewport.Width, m.id))
		m.viewport.GotoBottom()
	}
}

func (m *Model) flushHistory() {
	if len(m.pendingHistory) == 0 {
		return
	}
	m.messages = append(m.pendingHistory, m.messages...)
	for _, h := range m.pendingHistory {
		if h.CreatedAt > 0 && (h.CreatedAt < m.oldestTS || m.oldestTS == 0) {
			m.oldestTS = h.CreatedAt
		}
	}
	m.pendingHistory = nil
	m.viewport.SetContent(renderMessages(m.messages, m.viewport.Width, m.id))
}

func (m *Model) tryLoadHistory() {
	if m.historyLoading || m.historyEnd {
		return
	}
	if !m.viewport.AtTop() {
		return
	}
	m.historyLoading = true
	m.connector.Send(&protocol.Message{
		Type:    "command",
		FromID:  m.id,
		Content: fmt.Sprintf("/history %d 50", m.oldestTS),
	})
}
