package tui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"ZhatRoom/internal/protocol"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.winWidth = msg.Width
		m.winHeight = msg.Height
		m.viewport.Height = msg.Height - 1 - m.footerHeight()
		m.viewport.YPosition = 0
		m.input.Width = msg.Width - sidebarWidth - 4
		if !m.welcomeSent && m.viewport.Height > 0 {
			state := m.currentRoom()
			state.messages = append(state.messages, protocol.Message{
				Type:    "system",
				From:    "System",
				Content: fmt.Sprintf("Welcome to ZhatRoom! You are %s (%s)", m.nickname, m.id),
			})
			content, _, lineMap := renderMessages(state.messages, m.viewport.Width, m.id, -1)
			m.viewport.SetContent(content)
			m.visualLineMap = lineMap
			m.viewport.GotoBottom()
			m.welcomeSent = true
		}

	case tea.KeyMsg:
		switch m.mode {
		case InputMode:
			return m.updateInputMode(msg)
		case NormalMode:
			return m.updateNormalMode(msg)
		case SidebarMode:
			return m.updateSidebarMode(msg)
		}

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

// --- Input Mode ---

func (m *Model) updateInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	hints := m.matchingCommands()

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEsc:
		m.mode = NormalMode
		m.input.Blur()
		state := m.currentRoom()
		m.cursorMsgIdx = len(state.messages) - 1
		m.cursorSubLine = 0
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyTab:
		if len(hints) > 0 {
			idx := m.cmdIdx
			if idx < 0 || idx >= len(hints) {
				idx = 0
			}
			val := m.input.Value()
			if len(val) >= 6 && strings.ToLower(val[:6]) == "/room " {
				m.input.SetValue("/room " + hints[idx].name)
			} else {
				m.input.SetValue("/" + hints[idx].name)
			}
			m.input.CursorEnd()
			m.cmdIdx = -1
			return m, nil
		}

	case tea.KeyUp:
		if len(hints) > 0 {
			if m.cmdIdx <= 0 {
				m.cmdIdx = len(hints) - 1
			} else {
				m.cmdIdx--
			}
			return m, nil
		}

	case tea.KeyDown:
		if len(hints) > 0 {
			if m.cmdIdx >= len(hints)-1 {
				m.cmdIdx = 0
			} else {
				m.cmdIdx++
			}
			return m, nil
		}

	case tea.KeyEnter:
		if len(hints) > 0 && m.cmdIdx >= 0 && m.cmdIdx < len(hints) {
			val := m.input.Value()
			if len(val) >= 6 && strings.ToLower(val[:6]) == "/room " {
				m.input.SetValue("/room " + hints[m.cmdIdx].name)
			} else {
				m.input.SetValue("/" + hints[m.cmdIdx].name)
			}
			m.input.CursorEnd()
			m.cmdIdx = -1
			return m, nil
		}

		text := m.input.Value()
		m.input.SetValue("")
		m.cmdIdx = -1
		if m.winHeight > 0 {
			m.viewport.Height = m.winHeight - 1 - m.footerHeight()
		}

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
	m.cmdIdx = 0
	if m.winHeight > 0 {
		m.viewport.Height = m.winHeight - 1 - m.footerHeight()
	}
	return m, tea.Batch(cmds...)
}

// --- Normal Mode ---

func (m *Model) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i", "a":
		m.mode = InputMode
		m.cursorMsgIdx = -1
		m.cursorSubLine = 0
		m.input.Focus()
		return m, textinput.Blink

	case "h":
		m.mode = SidebarMode
		m.cursorMsgIdx = -1
		m.cursorSubLine = 0
		for i, r := range m.roomList {
			if r.ID == m.currentRoomID {
				m.sidebarCursor = i
				break
			}
		}
		return m, nil

	case "j":
		m.moveCursorDown()
		return m, nil

	case "k":
		m.moveCursorUp()
		return m, nil

	case "G":
		state := m.currentRoom()
		m.cursorMsgIdx = len(state.messages) - 1
		m.cursorSubLine = 0
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, nil

	case "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// updateViewportContent re-renders viewport content with cursor highlight.
func (m *Model) updateViewportContent() {
	state := m.currentRoom()
	cursorLine := m.cursorLinePos()
	content, _, lineMap := renderMessages(state.messages, m.viewport.Width, m.id, cursorLine)
	m.viewport.SetContent(content)
	m.visualLineMap = lineMap
}

// cursorLinePos returns the visual line index of the cursor.
func (m *Model) cursorLinePos() int {
	state := m.currentRoom()
	if m.cursorMsgIdx < 0 || m.cursorMsgIdx >= len(state.messages) {
		return -1
	}
	line := 0
	var prevTS int64
	for i := 0; i < m.cursorMsgIdx && i < len(state.messages); i++ {
		msg := state.messages[i]
		if msg.Type == "history_end" {
			continue
		}
		if msg.CreatedAt > 0 && prevTS > 0 && msg.CreatedAt-prevTS > 300 {
			line++ // timestamp separator
		}
		if msg.CreatedAt > 0 {
			prevTS = msg.CreatedAt
		}
		if msg.Type != "system" && line > 0 {
			line++ // blank separator
		}
		content := msg.Content
		if msg.Type == "system" {
			content = fmt.Sprintf("[SYSTEM]: %s", msg.Content)
		} else if msg.FromID == m.id {
			content = fmt.Sprintf("[You]: %s", msg.Content)
		} else {
			content = fmt.Sprintf("[%s]: %s", msg.From, msg.Content)
		}
		vLines := 1
		for _, part := range strings.Split(content, "\n") {
			if m.viewport.Width > 0 && len(part) > m.viewport.Width {
				vLines += (len(part) - 1) / m.viewport.Width
			}
		}
		line += vLines
	}
	return line + m.cursorSubLine
}

// moveCursorDown moves the cursor down one visual line.
func (m *Model) moveCursorDown() {
	state := m.currentRoom()
	msgs := state.messages
	if len(msgs) == 0 {
		return
	}
	if m.cursorMsgIdx < 0 {
		m.cursorMsgIdx = len(msgs) - 1
		m.cursorSubLine = 0
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return
	}
	if m.cursorMsgIdx >= len(msgs)-1 {
		m.viewport.GotoBottom()
		return
	}

	m.cursorSubLine++
	msg := msgs[m.cursorMsgIdx]
	if msg.Type == "history_end" {
		m.cursorMsgIdx++
		m.cursorSubLine = 0
	} else {
		content := msg.Content
		if msg.Type == "system" {
			content = fmt.Sprintf("[SYSTEM]: %s", msg.Content)
		} else if msg.FromID == m.id {
			content = fmt.Sprintf("[You]: %s", msg.Content)
		} else {
			content = fmt.Sprintf("[%s]: %s", msg.From, msg.Content)
		}
		vLines := 1
		for _, part := range strings.Split(content, "\n") {
			if m.viewport.Width > 0 && len(part) > m.viewport.Width {
				vLines += (len(part) - 1) / m.viewport.Width
			}
		}
		if m.cursorSubLine >= vLines {
			m.cursorMsgIdx++
			m.cursorSubLine = 0
		}
	}

	m.updateViewportContent()

	cursorLine := m.cursorLinePos()
	if cursorLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = cursorLine - m.viewport.Height + 1
	}
}

// moveCursorUp moves the cursor up one visual line.
func (m *Model) moveCursorUp() {
	state := m.currentRoom()
	msgs := state.messages
	if len(msgs) == 0 {
		return
	}
	if m.cursorMsgIdx < 0 {
		m.cursorMsgIdx = len(msgs) - 1
		m.cursorSubLine = 0
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return
	}
	if m.cursorMsgIdx >= len(msgs) {
		m.cursorMsgIdx = len(msgs) - 1
	}

	if m.cursorSubLine > 0 {
		m.cursorSubLine--
	} else if m.cursorMsgIdx > 0 {
		m.cursorMsgIdx--
		// Skip history_end markers
		for m.cursorMsgIdx > 0 && msgs[m.cursorMsgIdx].Type == "history_end" {
			m.cursorMsgIdx--
		}
		// Set to last sub-line of the previous message
		prevMsg := msgs[m.cursorMsgIdx]
		if prevMsg.Type == "history_end" {
			m.cursorSubLine = 0
		} else {
			content := prevMsg.Content
			if prevMsg.Type == "system" {
				content = fmt.Sprintf("[SYSTEM]: %s", prevMsg.Content)
			} else if prevMsg.FromID == m.id {
				content = fmt.Sprintf("[You]: %s", prevMsg.Content)
			} else {
				content = fmt.Sprintf("[%s]: %s", prevMsg.From, prevMsg.Content)
			}
			vLines := 1
			for _, part := range strings.Split(content, "\n") {
				if len(part) > m.viewport.Width {
					vLines += (len(part) - 1) / m.viewport.Width
				}
			}
			m.cursorSubLine = vLines - 1
		}
	} else {
		// At top, try loading history
		m.tryLoadHistory()
		return
	}

	m.updateViewportContent()

	cursorLine := m.cursorLinePos()
	if cursorLine < m.viewport.YOffset {
		m.viewport.YOffset = cursorLine
	}
}

// --- Sidebar Mode ---

func (m *Model) updateSidebarMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "l", "esc":
		m.mode = NormalMode
		state := m.currentRoom()
		m.cursorMsgIdx = len(state.messages) - 1
		m.cursorSubLine = 0
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, nil

	case "j":
		if m.sidebarCursor < len(m.roomList)-1 {
			m.sidebarCursor++
		}
		m.previewSidebarRoom()
		return m, nil

	case "k":
		if m.sidebarCursor > 0 {
			m.sidebarCursor--
		}
		m.previewSidebarRoom()
		return m, nil

	case "enter":
		// Formal join — send /room join to server
		m.commitSidebarRoom()
		m.mode = NormalMode
		return m, nil

	case "i":
		// Formal join + enter input mode
		m.commitSidebarRoom()
		m.mode = InputMode
		m.cursorMsgIdx = -1
		m.cursorSubLine = 0
		m.input.Focus()
		return m, textinput.Blink

	case "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// previewSidebarRoom switches viewport content locally (no server message).
func (m *Model) previewSidebarRoom() {
	if m.sidebarCursor < 0 || m.sidebarCursor >= len(m.roomList) {
		return
	}
	selected := m.roomList[m.sidebarCursor]
	if selected.ID == m.currentRoomID {
		return
	}
	// Local switch — no server roundtrip, no join/leave broadcast
	m.switchRoom(selected.ID)
	m.viewport.GotoBottom()
}

// commitSidebarRoom sends /room join to server for formal join.
func (m *Model) commitSidebarRoom() {
	if m.sidebarCursor < 0 || m.sidebarCursor >= len(m.roomList) {
		return
	}
	selected := m.roomList[m.sidebarCursor]
	if selected.ID == m.currentRoomID {
		return
	}
	joinMsg := &protocol.Message{
		Type:    "command",
		FromID:  m.id,
		Content: fmt.Sprintf("/room join %d", selected.ID),
	}
	if err := m.connector.Send(joinMsg); err != nil {
		m.err = err
	}
}

// --- Incoming message handling ---

func (m *Model) handleIncoming(msg *protocol.Message) {
	switch msg.Type {
	case "room_list":
		var rooms []protocol.RoomSummary
		if err := json.Unmarshal(msg.Data, &rooms); err != nil {
			return
		}
		m.roomList = rooms
		for _, r := range rooms {
			if _, ok := m.roomStates[r.ID]; !ok {
				m.roomStates[r.ID] = &RoomState{}
			}
		}
		return

	case "room_switch":
		roomID, err := strconv.ParseUint(msg.Content, 10, 64)
		if err != nil {
			return
		}
		m.switchRoom(uint(roomID))
		m.viewport.GotoBottom()
		state := m.currentRoom()
		if len(state.messages) == 0 && !state.historyLoading {
			state.historyLoading = true
			m.connector.Send(&protocol.Message{
				Type:    "command",
				FromID:  m.id,
				Content: "/history 0 50",
			})
		}
		return

	case "history":
		state := m.currentRoom()
		state.pendingHistory = append(state.pendingHistory, *msg)
		return

	case "history_end":
		state := m.currentRoom()
		m.flushHistory(state)
		state.historyLoading = false
		if len(state.pendingHistory) == 0 && !state.historyEnd {
			state.historyEnd = true
		}
		return

	default:
		state := m.currentRoom()
		if len(state.pendingHistory) > 0 {
			m.flushHistory(state)
		}
		atBottom := m.viewport.AtBottom()
		wasLast := m.cursorMsgIdx >= 0 && m.cursorMsgIdx >= len(state.messages)-1
		state.messages = append(state.messages, *msg)
		if msg.CreatedAt > 0 && (msg.CreatedAt < state.oldestTS || state.oldestTS == 0) {
			state.oldestTS = msg.CreatedAt
		}
		if msg.RoomID == m.currentRoomID || msg.RoomID == 0 {
			if wasLast {
				m.cursorMsgIdx = len(state.messages) - 1
			}
			m.updateViewportContent()
			if atBottom || wasLast {
				m.viewport.GotoBottom()
			}
		}
	}
}

func (m *Model) flushHistory(state *RoomState) {
	if len(state.pendingHistory) == 0 {
		return
	}
	firstLoad := !state.historyReceived
	state.messages = append(state.pendingHistory, state.messages...)
	for _, h := range state.pendingHistory {
		if h.CreatedAt > 0 && (h.CreatedAt < state.oldestTS || state.oldestTS == 0) {
			state.oldestTS = h.CreatedAt
		}
	}
	state.pendingHistory = nil
	state.historyReceived = true
	content, _, lineMap := renderMessages(state.messages, m.viewport.Width, m.id, -1)
	m.viewport.SetContent(content)
	m.visualLineMap = lineMap
	if firstLoad {
		m.viewport.GotoBottom()
	}
}

func (m *Model) tryLoadHistory() {
	state := m.currentRoom()
	if state.historyLoading || state.historyEnd {
		return
	}
	if !m.viewport.AtTop() {
		return
	}
	state.historyLoading = true
	m.connector.Send(&protocol.Message{
		Type:    "command",
		FromID:  m.id,
		Content: fmt.Sprintf("/history %d 50", state.oldestTS),
	})
}
