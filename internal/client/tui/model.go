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
	{"room", "create/join/delete/list 房间"},
	{"history", "加载历史消息"},
}

var roomSubCommands = []cmdEntry{
	{"create", "创建房间"},
	{"join", "加入房间"},
	{"delete", "删除房间"},
	{"list", "列出所有房间"},
}

type Mode int

const (
	InputMode  Mode = iota // typing messages
	NormalMode             // vim-style navigation
	SidebarMode            // browsing room list
)

type RoomState struct {
	messages       []protocol.Message
	pendingHistory []protocol.Message
	viewportPos    int
	historyLoading  bool
	historyEnd      bool
	historyReceived bool
	oldestTS        int64
}

type Model struct {
	connector *Connector
	viewport  viewport.Model
	input     textinput.Model
	nickname  string
	id        string
	ready     bool
	err       error
	winWidth  int
	winHeight int
	cmdIdx    int
	welcomeSent bool

	// Mode system
	mode Mode

	// Cursor for NormalMode line navigation
	cursorMsgIdx      int   // message index the cursor points to
	cursorVisualLine  int   // current visual line index in lineMap
	cursorSubLine     int   // sub-line within a multi-line message
	visualLineMap     []int // visual line index → message index

	// Room management
	roomStates     map[uint]*RoomState
	currentRoomID  uint
	currentRoomName string
	roomList       []protocol.RoomSummary
	sidebarCursor  int
}

func NewModel(id, nickname string, connector *Connector) *Model {
	ti := textinput.New()
	ti.Placeholder = "输入消息..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 40

	return &Model{
		connector:      connector,
		viewport:       viewport.New(0, 0),
		input:          ti,
		nickname:       nickname,
		id:             id,
		mode:           InputMode,
		roomStates:     map[uint]*RoomState{0: {}},
		currentRoomID:  0,
		currentRoomName: "lobby",
		roomList:       []protocol.RoomSummary{{ID: 0, Name: "lobby"}},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		waitForMessage(m.connector),
	)
}

func (m *Model) currentRoom() *RoomState {
	return m.roomStates[m.currentRoomID]
}

func (m *Model) switchRoom(roomID uint) {
	// Save current viewport position
	if state, ok := m.roomStates[m.currentRoomID]; ok {
		state.viewportPos = m.viewport.YOffset
	}

	m.currentRoomID = roomID

	// Update room name from roomList
	for _, r := range m.roomList {
		if r.ID == roomID {
			m.currentRoomName = r.Name
			break
		}
	}

	// Restore or initialize state
	state, ok := m.roomStates[roomID]
	if !ok {
		state = &RoomState{}
		m.roomStates[roomID] = state
	}

	content, _, lineMap := renderMessages(state.messages, m.viewport.Width, m.id, -1)
	m.viewport.SetContent(content)
	m.visualLineMap = lineMap
	m.cursorVisualLine = -1
	m.viewport.YOffset = state.viewportPos
}

func (m *Model) modeString() string {
	switch m.mode {
	case InputMode:
		return "input"
	case NormalMode:
		return "normal"
	case SidebarMode:
		return "sidebar"
	default:
		return "?"
	}
}
