package server

import (
	"ZhatRoom/internal/protocol"
	"fmt"
	"sync"
)

const LobbyID uint = 0

type Room struct {
	ID      uint
	Name    string
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewRoom(id uint, name string) *Room {
	return &Room{
		ID:      id,
		Name:    name,
		clients: make(map[string]*Client),
	}
}

func (r *Room) Join(c *Client) {
	r.mu.Lock()
	r.clients[c.ID] = c
	r.mu.Unlock()
	c.room = r
	fmt.Printf("[Room:%s]: %s joined\n", r.Name, c.Nickname)
}

func (r *Room) Leave(c *Client) {
	r.mu.Lock()
	delete(r.clients, c.ID)
	r.mu.Unlock()
	fmt.Printf("[Room:%s]: %s left\n", r.Name, c.Nickname)
}

func (r *Room) Broadcast(msg *protocol.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.clients {
		if err := c.Send(msg); err != nil {
			fmt.Printf("[Room:%s]: failed to send to %s: %v\n", r.Name, c.ID, err)
		}
	}
}

func (r *Room) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}
