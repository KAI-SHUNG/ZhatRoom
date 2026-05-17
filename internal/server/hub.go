package server

import (
	"ZhatRoom/internal/config"
	"ZhatRoom/internal/protocol"
	"encoding/json"
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
	rooms      map[uint]*Room
	maxClients int

	mu sync.RWMutex
}

func NewHub(cfg config.ServerConfig) *Hub {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic("failed to create snowflake node: " + err.Error())
	}

	lobby := NewRoom(LobbyID, "lobby")
	rooms := map[uint]*Room{LobbyID: lobby}

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

			lobby := h.rooms[LobbyID]
			lobby.Join(client)

			fmt.Printf("[Hub]: client %s registered\n", client.ID)
			go h.SendHistory(client, LobbyID, 50)
			go h.SendRoomList(client)

			lobby.Broadcast(systemMsg(fmt.Sprintf("%s 加入了聊天室", client.Nickname)))

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
				client.room.Broadcast(systemMsg(fmt.Sprintf("%s 离开了聊天室", client.Nickname)))
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

func (h *Hub) GetRoom(id uint) *Room {
	return h.rooms[id]
}

func (h *Hub) CreateRoom(name string, ownerID string) (*Room, error) {
	rm, err := h.store.CreateRoom(name, ownerID)
	if err != nil {
		return nil, err
	}
	room := NewRoom(rm.ID, rm.Name)
	h.rooms[rm.ID] = room
	return room, nil
}

func (h *Hub) JoinRoom(c *Client, roomID uint) (*Room, error) {
	room, ok := h.rooms[roomID]
	if !ok {
		// Try loading from DB
		rm, err := h.store.GetRoomByID(roomID)
		if err != nil {
			return nil, fmt.Errorf("room not found: %d", roomID)
		}
		room = NewRoom(rm.ID, rm.Name)
		h.rooms[rm.ID] = room
	}

	if c.room != nil {
		c.room.Broadcast(systemMsg(fmt.Sprintf("%s 离开了聊天室", c.Nickname)))
		c.room.Leave(c)
	}

	room.Join(c)
	room.Broadcast(systemMsg(fmt.Sprintf("%s 加入了聊天室", c.Nickname)))
	return room, nil
}

func (h *Hub) KickToLobby(c *Client) {
	if c.room != nil {
		c.room.Leave(c)
	}
	lobby := h.rooms[LobbyID]
	lobby.Join(c)
}

func (h *Hub) SendHistory(c *Client, roomID uint, limit int) {
	msgs, err := h.store.GetMessages(roomID, limit, 0)
	if err != nil {
		c.Send(systemMsg("Failed to load history"))
		return
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		msgs[i].Type = "history"
		if err := c.Send(&msgs[i]); err != nil {
			return
		}
	}
	c.Send(&protocol.Message{Type: "history_end"})
}

func (h *Hub) buildRoomList() []protocol.RoomSummary {
	rooms, err := h.store.ListRooms()
	if err != nil {
		fmt.Printf("[Hub]: failed to list rooms: %v\n", err)
		return nil
	}
	var summaries []protocol.RoomSummary
	for _, r := range rooms {
		count := 0
		if rm, ok := h.rooms[r.ID]; ok {
			count = rm.Count()
		}
		summaries = append(summaries, protocol.RoomSummary{
			ID:      r.ID,
			Name:    r.Name,
			Members: count,
		})
	}
	return summaries
}

func (h *Hub) SendRoomList(c *Client) {
	summaries := h.buildRoomList()
	data, _ := json.Marshal(summaries)
	c.Send(&protocol.Message{Type: "room_list", Data: data})
}

func (h *Hub) BroadcastRoomList() {
	summaries := h.buildRoomList()
	data, _ := json.Marshal(summaries)
	msg := &protocol.Message{Type: "room_list", Data: data}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients {
		c.Send(msg)
	}
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
