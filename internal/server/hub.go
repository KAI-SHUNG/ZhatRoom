package server

import (
	"ZhatRoom/internal/config"
	"ZhatRoom/internal/protocol"
	"fmt"
	"net"
	"sync"

	"github.com/bwmarrin/snowflake"
)

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *protocol.Message

	store      *Storage
	snowflake  *snowflake.Node
	commands   *CommandRegistry
	rooms      map[string]*Room
	maxClients int

	mu sync.RWMutex
}

func NewHub(cfg config.ServerConfig) *Hub {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic("failed to create snowflake node: " + err.Error())
	}

	lobby := NewRoom("lobby")
	rooms := map[string]*Room{"lobby": lobby}

	return &Hub{
		clients:    make(map[string]*Client, cfg.MaxClients),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *protocol.Message),
		store:      InitDB(cfg.DB.DSN()),
		snowflake:  node,
		commands:   NewCommandRegistry(),
		rooms:      rooms,
		maxClients: cfg.MaxClients,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client, ok := <-h.register:
			if !ok || client == nil {
				return
			}
			h.mu.Lock()
			if old, ok := h.clients[client.ID]; ok {
				old.Conn.Close()
			}
			h.clients[client.ID] = client
			h.mu.Unlock()

			lobby := h.rooms["lobby"]
			lobby.Join(client)

			fmt.Printf("[Hub]: client %s registered\n", client.ID)

		case client, ok := <-h.unregister:
			if !ok || client == nil {
				return
			}
			h.mu.Lock()
			if cur, ok := h.clients[client.ID]; ok && cur == client {
				delete(h.clients, client.ID)
			}
			h.mu.Unlock()

			if client.room != nil {
				client.room.Leave(client)
			}
			fmt.Printf("[Hub]: client %s unregistered\n", client.ID)

		case msg, ok := <-h.broadcast:
			if !ok || msg == nil {
				return
			}
			h.processMessage(msg)
		}
	}
}

func (h *Hub) Validate(uid string) bool {
	exist, err := h.store.UserExists(uid)
	if err != nil {
		fmt.Printf("[Hub]: validate error: %v\n", err)
		return false
	}
	return exist
}

func (h *Hub) HandleNewConn(conn net.Conn, id string, nickname string) {
	h.mu.RLock()
	currentClients := len(h.clients)
	h.mu.RUnlock()
	if currentClients >= h.maxClients {
		fmt.Fprint(conn, "Reached Max Online Clients. Connection closed.")
		conn.Close()
		return
	}

	exist, err := h.store.UserExists(id)
	if err != nil {
		fmt.Printf("[Hub]: database error: %v\n", err)
		conn.Close()
		return
	}
	if !exist {
		if err := h.store.NewUser(id, nickname); err != nil {
			fmt.Printf("[Hub]: failed to create user: %v\n", err)
			conn.Close()
			return
		}
	}

	c := &Client{
		Hub:      h,
		Conn:     conn,
		ID:       id,
		Nickname: nickname,
	}
	h.register <- c
	go c.Read()
}

func (h *Hub) Shutdown() {
	h.mu.Lock()
	clients := h.clients
	h.clients = make(map[string]*Client)
	h.mu.Unlock()

	for _, c := range clients {
		c.Conn.Close()
	}

	close(h.register)
	close(h.unregister)
	close(h.broadcast)
	h.store.Close()
}
