package server

import (
	"ZhatRoom/internal/protocol"
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	Hub      *Hub
	Conn     net.Conn
	ID       string
	Nickname string
}

/** Send message to client
 * * Serialize message to JSON and write
 */
func (c *Client) Send(msg *protocol.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("[Client %s]: failed to serialize message: %v", c.ID, err)
	}
	_, err = c.Conn.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("[Client %s]: failed to send message: %v", c.ID, err)
	}
	return nil
}

/** Read message from client
 * * Parse JSON to message and queue in broadcast channel
 */
func (c *Client) Read() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	reader := bufio.NewReader(c.Conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("[Client %s]: Connection closed...", c.ID)
			return
		}
		msg, err := protocol.FromJSON(line)
		if err != nil {
			fmt.Printf("[Client %s]: failed to parse message: %v\n", c.ID, err)
			return
		}
		fmt.Printf("[Client %s]: Received message: %+v\n", c.ID, *msg)

		c.Hub.broadcast <- msg
	}
}
