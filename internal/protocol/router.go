package protocol

import "fmt"

type ConnContext interface {
	ID() string            // 获取连接的唯一标识
	Send(m *Message) error // 给这个连接发送消息
}

type Router struct {
	handlers map[string]func(*Message) error
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]func(*Message) error),
	}
}

func (r *Router) Register(messageType string, handler func(*Message) error) {
	r.handlers[messageType] = handler
}

func (r *Router) Dispatch(msg *Message) error {
	handler, ok := r.handlers[msg.Type]
	if !ok {
		return fmt.Errorf("no handler registered for message type: %s", msg.Type)
	}

	return handler(msg)
}

func Bind[T AuthMessage | ChatMessage | CmdMessage | SystemMessage](
	logic func(*T) error) func(*Message) error {
	return func(m *Message) error {
		payload, err := GetPayload[T](m)
		if err != nil {
			return err
		}
		return logic(payload)
	}
}
