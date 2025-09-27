package models

import "time"

type Chat struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Members   []string  `json:"members"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	Type      string `json:"type"`
	ChatID    int    `json:"chat_id,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Content   string `json:"content,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`

	ChatName  string   `json:"chat_name,omitempty"`
	Members   []string `json:"members,omitempty"`
	CreatedBy string   `json:"created_by,omitempty"`
	Key       []byte   `json:"key"`
}

type KeyMassage struct {
	Content []byte
	Key     []byte
}
