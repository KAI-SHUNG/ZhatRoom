package tui

import (
	"ZhatRoom/internal/protocol"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	connector      *Connector
	messages       []protocol.Message
	pendingHistory []protocol.Message
	viewport       viewport.Model
	input          textinput.Model
	nickname       string
	id             string
	currentRoom    string
	ready          bool
	err            error
	welcomeSent    bool
	historyLoading bool
	historyEnd     bool
	oldestTS       int64
}

func NewModel(id, nickname string, connector *Connector) *Model {
	ti := textinput.New()
	ti.Placeholder = "输入消息..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 40

	return &Model{
		connector:   connector,
		messages:    []protocol.Message{},
		viewport:    viewport.New(0, 0),
		input:       ti,
		nickname:    nickname,
		id:          id,
		currentRoom: "lobby",
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForMessage(m.connector),
	)
}
