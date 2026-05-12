package protocol

import (
	"encoding/json"
)

type Message struct {
	ID        string `json:"uid" gorm:"primaryKey"`
	Type      string `json:"type"`
	From      string `json:"from"`
	FromID    string `json:"from_id"`
	Room      string `json:"room" gorm:"index"`
	CreatedAt int64  `json:"ts" gorm:"autoCreateTime"`
	Content   string `json:"content"`
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

type User struct {
	ID        string `json:"id" gorm:"primaryKey"`
	Nickname  string `json:"nickname" gorm:"not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}
