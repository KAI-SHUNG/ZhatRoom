package tui

import (
	"ZhatRoom/internal/protocol"
	"bufio"
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
)

type incomingMsg struct {
	msg *protocol.Message
}

type errMsg struct {
	err error
}

type Connector struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewConnector(socketPath, id, nickname string) (*Connector, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintf(conn, "%s,%s\n", id, nickname); err != nil {
		conn.Close()
		return nil, err
	}
	return &Connector{
		conn:   conn,
		reader: bufio.NewReader(conn),
	}, nil
}

func (c *Connector) Send(msg *protocol.Message) error {
	data, err := msg.ToJSON()
	if err != nil {
		return err
	}
	_, err = c.conn.Write(append(data, '\n'))
	return err
}

func (c *Connector) Close() error {
	return c.conn.Close()
}

func waitForMessage(c *Connector) func() tea.Msg {
	return func() tea.Msg {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return errMsg{err}
		}
		msg, err := protocol.FromJSON(line)
		if err != nil {
			return errMsg{err}
		}
		return incomingMsg{msg}
	}
}
