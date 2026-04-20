package protocol

import (
	"encoding/json"
)

type Message struct {
	Type      string          `json:"type"`
	TimeStamp int64           `json:"ts"`
	Payload   json.RawMessage `json:"payload"`
}
type AuthMessage struct {
	From     string `json:"from"`
	Username string `json:"username"`
	Token    string `json:"token"`
}
type ChatMessage struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	Content  string `json:"content"`
}
type CmdMessage struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	Command  string `json:"command"`
}
type SystemMessage struct {
	Info string `json:"info"`
}

func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func GetPayload[T AuthMessage | ChatMessage | CmdMessage | SystemMessage](
	m *Message) (*T, error) {
	var payload T
	err := json.Unmarshal(m.Payload, &payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func PayloadToJSON[T AuthMessage | ChatMessage | CmdMessage | SystemMessage](
	message *Message,
	payload *T) error {
	if data, err := json.Marshal(payload); err != nil {
		return err
	} else {
		message.Payload = json.RawMessage(data)
		return nil
	}
}
