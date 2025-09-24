package models

import "time"

type Chat struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Members   []string  `json:"members"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
