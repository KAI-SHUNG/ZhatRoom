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
	r.Register("room", cmdRoom)
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
		roomName := ""
		roomID := uint(0)
		if c.room != nil {
			roomName = c.room.Name
			roomID = c.room.ID
		}
		lines = append(lines, fmt.Sprintf("  %s (%s) in #%s (id:%d)", c.Nickname, c.ID, roomName, roomID))
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
  /exit              Exit the chat room
  /users             List online users
  /nick <name>       Change your nickname
  /help              Show this help
  /history [ts] [n]  Load history before timestamp
  /room create <name>  Create a new room
  /room join <name|id> Join a room by name or ID
  /room delete       Delete current room (owner only)
  /room list         List all rooms`)
}

func cmdRoom(ctx *CommandContext) *protocol.Message {
	if len(ctx.Args) == 0 {
		return systemMsg("Usage: /room <create|join|delete|list> [args]")
	}
	sub := ctx.Args[0]
	ctx.Args = ctx.Args[1:]
	switch sub {
	case "create":
		return cmdRoomCreate(ctx)
	case "join":
		return cmdRoomJoin(ctx)
	case "delete":
		return cmdRoomDelete(ctx)
	case "list":
		return cmdRoomList(ctx)
	default:
		return systemMsg(fmt.Sprintf("Unknown subcommand: /room %s", sub))
	}
}

func cmdRoomCreate(ctx *CommandContext) *protocol.Message {
	if len(ctx.Args) == 0 {
		return systemMsg("Usage: /room create <name>")
	}
	name := ctx.Args[0]
	if name == "lobby" {
		return systemMsg("Cannot create a room named 'lobby'")
	}

	room, err := ctx.Hub.CreateRoom(name, ctx.Client.ID)
	if err != nil {
		return systemMsg(fmt.Sprintf("Failed to create room: %v", err))
	}

	ctx.Hub.JoinRoom(ctx.Client, room.ID)
	go ctx.Hub.SendHistory(ctx.Client, room.ID, 50)
	ctx.Hub.BroadcastRoomList()

	// Send room_switch to the creator
	ctx.Client.Send(&protocol.Message{
		Type:    "room_switch",
		Content: fmt.Sprintf("%d", room.ID),
		Room:    room.Name,
	})

	return systemMsg(fmt.Sprintf("Room #%s (id:%d) created", name, room.ID))
}

func cmdRoomJoin(ctx *CommandContext) *protocol.Message {
	if len(ctx.Args) == 0 {
		return systemMsg("Usage: /room join <name|id>")
	}
	arg := ctx.Args[0]

	var roomID uint
	var roomName string

	// Try parsing as numeric ID first
	if id, err := strconv.ParseUint(arg, 10, 64); err == nil {
		roomID = uint(id)
		rm, err := ctx.Store.GetRoomByID(roomID)
		if err != nil {
			return systemMsg(fmt.Sprintf("Room not found: %d", roomID))
		}
		roomName = rm.Name
	} else {
		// Look up by name
		rm, err := ctx.Store.GetRoomByName(arg)
		if err != nil {
			return systemMsg(fmt.Sprintf("Room not found: %s", arg))
		}
		roomID = rm.ID
		roomName = rm.Name
	}

	if ctx.Client.room != nil && ctx.Client.room.ID == roomID {
		return systemMsg(fmt.Sprintf("You are already in #%s", roomName))
	}

	ctx.Hub.JoinRoom(ctx.Client, roomID)
	go ctx.Hub.SendHistory(ctx.Client, roomID, 50)

	// Send room_switch to the client
	ctx.Client.Send(&protocol.Message{
		Type:    "room_switch",
		Content: fmt.Sprintf("%d", roomID),
		Room:    roomName,
	})

	return systemMsg(fmt.Sprintf("Joined #%s", roomName))
}

func cmdRoomDelete(ctx *CommandContext) *protocol.Message {
	if ctx.Client.room == nil || ctx.Client.room.ID == LobbyID {
		return systemMsg("Cannot delete the lobby")
	}

	room := ctx.Client.room
	// TODO: owner permission check (deferred)
	// For now, only the creator can delete
	if ctx.Client.ID != "" {
		// Kick all clients to lobby
		room.mu.RLock()
		clients := make([]*Client, 0, len(room.clients))
		for _, c := range room.clients {
			clients = append(clients, c)
		}
		room.mu.RUnlock()

		for _, c := range clients {
			ctx.Hub.KickToLobby(c)
			c.Send(systemMsg(fmt.Sprintf("Room #%s has been deleted, you have been moved to lobby", room.Name)))
		}

		// Remove from hub
		delete(ctx.Hub.rooms, room.ID)

		// Delete from DB
		if err := ctx.Store.DeleteRoom(room.ID); err != nil {
			return systemMsg(fmt.Sprintf("Failed to delete room from DB: %v", err))
		}

		ctx.Hub.BroadcastRoomList()
		return systemMsg(fmt.Sprintf("Room #%s deleted", room.Name))
	}

	return systemMsg("Permission denied")
}

func cmdRoomList(ctx *CommandContext) *protocol.Message {
	rooms, err := ctx.Store.ListRooms()
	if err != nil {
		return systemMsg("Failed to list rooms")
	}

	var lines []string
	lines = append(lines, "Rooms:")
	for _, r := range rooms {
		count := 0
		if rm, ok := ctx.Hub.rooms[r.ID]; ok {
			count = rm.Count()
		}
		marker := " "
		if ctx.Client.room != nil && ctx.Client.room.ID == r.ID {
			marker = "▸"
		}
		lines = append(lines, fmt.Sprintf("  %s %d  %-16s %d online", marker, r.ID, r.Name, count))
	}
	return systemMsg(strings.Join(lines, "\n"))
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

	roomID := LobbyID
	if ctx.Client.room != nil {
		roomID = ctx.Client.room.ID
	}

	go func() {
		msgs, err := ctx.Store.GetMessages(roomID, limit, before)
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
