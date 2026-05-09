package server

import (
	"ZhatRoom/internal/protocol"
	"fmt"
)

// processMessage handles ID generation, persistence, and routing.
func (h *Hub) processMessage(msg *protocol.Message) {
	msg.ID = h.snowflake.Generate().String()

	if err := h.store.NewMessage(*msg); err != nil {
		fmt.Printf("[Hub]: failed to save message: %v\n", err)
		return
	}

	switch msg.Type {
	case "chat":
		h.mu.RLock()
		client := h.clients[msg.FromID]
		h.mu.RUnlock()

		if client == nil {
			return
		}
		client.room.Broadcast(msg)

	case "command":
		h.mu.RLock()
		client := h.clients[msg.FromID]
		h.mu.RUnlock()

		if client == nil {
			return
		}

		name, args := parseCommand(msg.Content)
		ctx := &CommandContext{
			Hub:    h,
			Client: client,
			Args:   args,
			Store:  h.store,
		}
		if resp := h.commands.Dispatch(name, ctx); resp != nil {
			if err := client.Send(resp); err != nil {
				fmt.Printf("[Hub]: failed to send command response to %s: %v\n", client.ID, err)
			}
		}
	}
}

// parseCommand splits "/exit arg1 arg2" into ("exit", ["arg1", "arg2"]).
func parseCommand(content string) (string, []string) {
	// content starts with "/"
	name := content[1:] // strip leading /
	parts := splitArgs(name)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func splitArgs(s string) []string {
	var args []string
	var current []rune
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if len(current) > 0 {
				args = append(args, string(current))
				current = nil
			}
		default:
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		args = append(args, string(current))
	}
	return args
}
