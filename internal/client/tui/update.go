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
		if !m.welcomeSent {
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
			m.viewport.HalfViewUp()
			return m, nil

		case tea.KeyPgDown:
			m.viewport.HalfViewDown()
			return m, nil
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case incomingMsg:
		m.messages = append(m.messages, *msg.msg)
		m.viewport.SetContent(renderMessages(m.messages, m.viewport.Width, m.id))
		m.viewport.GotoBottom()
		cmds = append(cmds, waitForMessage(m.connector))

	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}
