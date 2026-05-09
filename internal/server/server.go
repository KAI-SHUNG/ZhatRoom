package server

import (
	"ZhatRoom/internal/config"
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type Server struct {
	ln  net.Listener
	hub *Hub
	cfg config.ServerConfig
	wg  sync.WaitGroup
}

func NewServer(cfg config.ServerConfig) *Server {
	return &Server{
		hub: NewHub(cfg),
		cfg: cfg,
	}
}

func (s *Server) Listen() error {
	if err := os.Remove(s.cfg.Socket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove socket: %w", err)
	}
	ln, err := net.Listen("unix", s.cfg.Socket)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.cfg.Socket, err)
	}
	s.ln = ln
	fmt.Printf("[Server]: listening on %s\n", s.cfg.Socket)
	return nil
}

func (s *Server) AcceptLoop() {
	go s.hub.Run()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					fmt.Println("[Server]: shutting down, stop accepting.")
					return
				}
				fmt.Printf("[Server]: accept error: %v\n", err)
				continue
			}
			s.handleConn(conn)
		}
	}()
}

func (s *Server) handleConn(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Printf("[Server]: read error: %v\n", err)
		conn.Close()
		return
	}
	msg := strings.TrimSpace(string(line))

	// validate protocol: validate,<uid>
	if uid, ok := strings.CutPrefix(msg, "validate,"); ok {
		if s.hub.Validate(uid) {
			fmt.Fprint(conn, "ok\n")
		} else {
			fmt.Fprint(conn, "denied\n")
		}
		conn.Close()
		return
	}

	// handshake: id,nickname
	parts := strings.Split(msg, ",")
	if len(parts) != 2 {
		fmt.Println("[Server]: invalid handshake format")
		conn.Close()
		return
	}
	id, nickname := parts[0], parts[1]
	fmt.Printf("[Server]: connection from id=%s, nickname=%s\n", id, nickname)
	s.hub.HandleNewConn(conn, id, nickname)
}

func (s *Server) Shutdown() {
	fmt.Println("[Server]: shutting down...")
	s.ln.Close()
	s.wg.Wait()
	s.hub.Shutdown()
}
