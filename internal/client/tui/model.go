package tui

import (
	"ZhatRoom/internal/protocol"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type cmdEntry struct {
	name string
	desc string
}

var builtinCommands = []cmdEntry{
	{"exit", "退出聊天室"},
	{"users", "查看在线用户"},
	{"nick", "修改昵称"},
	{"help", "显示帮助"},
	{"join", "切换房间"},
	{"history", "加载历史消息"},
}

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
	historyLoading  bool
	historyEnd      bool
	historyReceived bool
	oldestTS        int64
	winWidth        int
	winHeight       int
	cmdIdx          int // selected index in command hints, -1 = none
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
