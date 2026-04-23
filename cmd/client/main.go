package main

import (
	"ZhatRoom/internal/protocol"
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	/** Parse command-line arguments
	 * TODO: TRY better way to login (DB check, etc.)
	 */
	id := flag.String("id", "001", "ZhatRoom Id")
	nickname := flag.String("usr", "Anonymous", "ZhatRoom nickname")
	flag.Parse()

	/** Handle graceful shutdown
	 * * Listen for TERMINATE signal (Ctrl+C)
	 * * and manual exit command
	 */
	exitChan := make(chan bool, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\nReceived shutdown signal, exiting...")
		exitChan <- true
	}()

	/** Socket connection
	 * TODO: HARD coded socket path
	 */
	socketPath := "/tmp/zhatroom.sock"
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Printf("dial error: %v\n", err)
		return
	}
	fmt.Printf("Connected to server at %s\n", socketPath)

	/**
	 * TODO: Better login process
	 * * Send id and nickname to server
	 */
	if _, err := fmt.Fprintf(conn, "%s,%s\n", *id, *nickname); err != nil {
		fmt.Printf("write error: %v\n", err)
		conn.Close()
		return
	}

	/**
	 * * Listen for server broadcast
	 * TODO: Handle different message types (chat, system, etc.)
	 */
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				fmt.Printf("read error: %v\n", err)
				return
			}
			msg, err := protocol.FromJSON(line)
			if err != nil {
				fmt.Printf("failed to parse message: %v\n", err)
				return
			}

			fmt.Printf("[%s]: %s\n", msg.From, msg.Content)
		}
	}()

	/**
	 * * Input loop for sending messages
	 * TODO: Bottom line input, colored print
	 */
	fmt.Println("已进入聊天室，直接输入内容按回车发送：")
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			}
			msg := &protocol.Message{
				Type:      "chat",
				From:      *nickname,
				FromID:    *id,
				CreatedAt: time.Now().Unix(),
				Content:   line,
			}
			if line[0] == '/' {
				msg.Type = "command"
			}

			data, err := msg.ToJSON()
			if err != nil {
				fmt.Printf("消息序列化失败: %v\n", err)
				continue
			}
			_, err = conn.Write(append(data, '\n'))
			if err != nil {
				fmt.Printf("发送失败: %v\n", err)
			}
		}
	}()

	/** Graceful shutdown
	 * * Block and listen for exit signal
	 * * Clean up connection
	 */
	<-exitChan
	fmt.Print("Closing connection...\n")
	conn.Close()
}
