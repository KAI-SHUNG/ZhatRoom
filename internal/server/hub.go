package server

import (
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

	db *Storage

	mu sync.RWMutex
}

/** Create a new Hub instance
 * * Return pointer to Hub
 */
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client, MAX_ONLINE_CLIENTS),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *protocol.Message),
		db:         InitDB(),
	}
}

/** Main loop for Hub
 * * Listen for channals
 */
func (h *Hub) Run() {
	for {
		select {
		/**
		 * * New client registration
		 */
		case client, ok := <-h.register:
			/**
			 * * Check for channel closure and nil client
			 */
			if !ok {
				fmt.Println("[Hub]: Register channel closed, stopping hub.")
				return
			}
			if client == nil {
				fmt.Println("[Hub]: Received nil client on register channel, ignoring.")
				return
			}

			h.mu.Lock()
			h.clients[client.ID] = client
			fmt.Printf("[Hub]: Client %s registered\n", client.ID)
			h.mu.Unlock()
		/**
		 * * Client unregistration
		 */
		case client, ok := <-h.unregister:
			/**
			 * * Check for channel closure and nil client
			 */
			if !ok {
				fmt.Println("[Hub]: Unregister channel closed, stopping hub.")
				return
			}
			if client == nil {
				fmt.Println("[Hub]: Received nil client on unregister channel, ignoring.")
				return
			}

			h.mu.Lock()
			delete(h.clients, client.ID)
			fmt.Printf("[Hub]: Client %s unregistered\n", client.ID)
			h.mu.Unlock()
		/**
		 * * Messages for broadcast
		 */
		case msg, ok := <-h.broadcast:
			/**
			 * * Check for channel closure and nil message
			 */
			if !ok {
				fmt.Println("[Hub]: Broadcast channel closed, stopping hub.")
				return
			}
			if msg == nil {
				fmt.Println("[Hub]: Received nil message on broadcast channel, ignoring.")
				return
			}

			// Generate unique ID for message
			node, err := snowflake.NewNode(1)
			if err != nil {
				fmt.Printf("[Hub]: failed to create snowflake node: %v\n", err)
				continue
			}
			h.mu.RLock()
			client := h.clients[msg.FromID]
			h.mu.RUnlock()

			msg.ID = node.Generate().String()
			// Save message to database
			if err := h.db.NewMessage(*msg); err != nil {
				fmt.Printf("[Hub]: failed to save message to database: %v\n", err)
				continue
			}
			switch msg.Type {
			case "chat":
				// Broadcast message to all clients
				h.mu.RLock()
				fmt.Printf("[Hub]: Broadcasting message: %+v\n", *msg)
				for _, client := range h.clients {
					err := client.Send(msg)
					if err != nil {
						fmt.Printf("[Hub]: failed to send message to client id %s: %v\n", client.ID, err)
					}
					fmt.Printf("[Hub]: Message sent to client: %s\n", client.ID)
				}
				h.mu.RUnlock()

				// TODO: maybe Encapsulate to HandleCommand
			case "command":
				if msg.Content == "/exit" {
					fmt.Printf("[Hub]: Received exit command from client id %s\n", msg.FromID)
					go func(c *Client) {
						h.unregister <- c
						c.Conn.Close()
					}(client)
				}
			}
		}
	}
}

/** Handle new client connection
 * * Check for max clients and user existence
 */
func (h *Hub) HandleNewConn(conn net.Conn, id string, nickname string) {
	/**
	 * * Use lock to safely check current number of clients
	 */
	h.mu.RLock()
	currentClients := len(h.clients)
	h.mu.RUnlock()
	if currentClients >= MAX_ONLINE_CLIENTS {
		fmt.Fprint(conn, "Reached Max Online Clients. Connection closed.")
		conn.Close()
		return
	}
	/**
	 * * Check if user already exists in database
	 * * if not create new user
	 */
	exist, err := h.db.UserExists(id)
	if err != nil {
		fmt.Printf("[Hub]: database error: %v\n", err)
		conn.Close()
		return
	}
	if !exist {
		if err := h.db.NewUser(id, nickname); err != nil {
			fmt.Printf("[Hub]: failed to create user: %v\n", err)
			conn.Close()
			return
		}
	}
	/**
	 * * New online client to register
	 */
	Client := &Client{
		Hub:      h,
		Conn:     conn,
		ID:       id,
		Nickname: nickname,
	}
	h.register <- Client
	/**
	 * * Start goroutine to read messages from client
	 */
	go Client.Read()
}

/** Shutdown the Hub and clean up resources
 * * Close all client connections, close channels, and close database connection
 */
func (h *Hub) Shutdown() {
	h.mu.Lock()
	clients := h.clients
	h.clients = make(map[string]*Client)
	h.mu.Unlock()

	for _, client := range clients {
		fmt.Printf("[Hub]: Closing connection for client %s\n", client.ID)
		client.Conn.Close()
	}

	fmt.Print("[Hub]: Closing register\n")
	close(h.register)
	fmt.Print("[Hub]: Closing unregister\n")
	close(h.unregister)
	fmt.Print("[Hub]: Closing broadcast\n")
	close(h.broadcast)
	fmt.Print("[Hub]: Closing database connection\n")
	h.db.Close()
}
