package server

import (
	"ZhatRoom/internal/protocol"
	"fmt"
	"strconv"
	"strings"
)

type CommandContext struct {
	Hub    *Hub
	Client *Client
	Args   []string
	Store  *Storage
}

type CommandHandler func(ctx *CommandContext) *protocol.Message

type CommandRegistry struct {
	handlers map[string]CommandHandler
}

func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{handlers: make(map[string]CommandHandler)}
	r.registerBuiltin()
	return r
}

func (r *CommandRegistry) Register(name string, handler CommandHandler) {
	r.handlers[name] = handler
}

func (r *CommandRegistry) Dispatch(name string, ctx *CommandContext) *protocol.Message {
	handler, ok := r.handlers[name]
	if !ok {
		return systemMsg(fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", name))
	}
	return handler(ctx)
}

func (r *CommandRegistry) registerBuiltin() {
	r.Register("exit", cmdExit)
	r.Register("users", cmdUsers)
	r.Register("nick", cmdNick)
	r.Register("help", cmdHelp)
	r.Register("join", cmdJoin)
	r.Register("history", cmdHistory)
}

func systemMsg(content string) *protocol.Message {
	return &protocol.Message{
		Type:    "system",
		From:    "System",
		Content: content,
	}
}

func cmdExit(ctx *CommandContext) *protocol.Message {
	go func() {
		ctx.Hub.unregister <- ctx.Client
		ctx.Client.Conn.Close()
	}()
	return nil
}

func cmdUsers(ctx *CommandContext) *protocol.Message {
	ctx.Hub.mu.RLock()
	defer ctx.Hub.mu.RUnlock()

	var lines []string
	lines = append(lines, fmt.Sprintf("Online users (%d):", len(ctx.Hub.clients)))
	for _, c := range ctx.Hub.clients {
		lines = append(lines, fmt.Sprintf("  %s (%s) in #%s", c.Nickname, c.ID, c.room.Name))
	}
	return systemMsg(strings.Join(lines, "\n"))
}

func cmdNick(ctx *CommandContext) *protocol.Message {
	if len(ctx.Args) == 0 {
		return systemMsg("Usage: /nick <new_nickname>")
	}
	old := ctx.Client.Nickname
	ctx.Client.Nickname = ctx.Args[0]
	return systemMsg(fmt.Sprintf("Nickname changed: %s → %s", old, ctx.Args[0]))
}

func cmdHelp(ctx *CommandContext) *protocol.Message {
	return systemMsg(`Available commands:
  /exit          Exit the chat room
  /users         List online users
  /nick <name>   Change your nickname
  /help          Show this help
  /history [ts] [n] Load history before timestamp
  /join <room>   Switch room`)
}

func cmdJoin(ctx *CommandContext) *protocol.Message {
	if len(ctx.Args) == 0 {
		return systemMsg("Usage: /join <room>")
	}
	roomName := ctx.Args[0]

	oldRoom := ctx.Client.room
	if oldRoom != nil {
		oldRoom.Leave(ctx.Client)
	}

	room := ctx.Hub.GetOrCreateRoom(roomName)
	room.Join(ctx.Client)

	go ctx.Hub.SendHistory(ctx.Client, roomName, 50)

	return systemMsg(fmt.Sprintf("Joined #%s", roomName))
}

func cmdHistory(ctx *CommandContext) *protocol.Message {
	var before int64
	limit := 50

	if len(ctx.Args) >= 1 {
		if v, err := strconv.ParseInt(ctx.Args[0], 10, 64); err == nil {
			before = v
		}
	}
	if len(ctx.Args) >= 2 {
		if v, err := strconv.Atoi(ctx.Args[1]); err == nil {
			limit = v
		}
	}
	if limit > 200 {
		limit = 200
	}

	roomName := "lobby"
	if ctx.Client.room != nil {
		roomName = ctx.Client.room.Name
	}

	go func() {
		msgs, err := ctx.Store.GetMessages(roomName, limit, before)
		if err != nil {
			ctx.Client.Send(systemMsg("Failed to load history"))
			return
		}
		if len(msgs) == 0 {
			ctx.Client.Send(&protocol.Message{Type: "history_end"})
			return
		}
		for i := len(msgs) - 1; i >= 0; i-- {
			msgs[i].Type = "history"
			if err := ctx.Client.Send(&msgs[i]); err != nil {
				return
			}
		}
	}()

	return nil
}
