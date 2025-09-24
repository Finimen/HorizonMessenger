package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  string
	ChatIDs map[string]bool
}

type Hub struct {
	Clients    map[string]*Client
	ChatRooms  map[string]map[string]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Mutex      sync.RWMutex
	Logger     *slog.Logger
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		ChatRooms:  make(map[string]map[string]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Logger:     logger,
	}
}

type Message struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Content   string `json:"content,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`

	ChatName  string   `json:"chat_name,omitempty"`
	Members   []string `json:"members,omitempty"`
	CreatedBy string   `json:"created_by,omitempty"`
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			h.Clients[client.UserID] = client
			if client.ChatIDs == nil {
				client.ChatIDs = make(map[string]bool)
			}
			h.Mutex.Unlock()
			h.Logger.Info("Client registered", "userID", client.UserID)

		case client := <-h.Unregister:
			h.Mutex.Lock()
			delete(h.Clients, client.UserID)
			for chatID := range client.ChatIDs {
				if users, ok := h.ChatRooms[chatID]; ok {
					delete(users, client.UserID)
					if len(users) == 0 {
						delete(h.ChatRooms, chatID)
					}
				}
			}
			close(client.Send)
			h.Mutex.Unlock()
			h.Logger.Info("Client unregistered", "userID", client.UserID)

		case message := <-h.Broadcast:
			h.Mutex.RLock()

			switch message.Type {
			case "join_chat":
				if h.ChatRooms[message.ChatID] == nil {
					h.ChatRooms[message.ChatID] = make(map[string]bool)
				}
				h.ChatRooms[message.ChatID][message.Sender] = true

				if client, ok := h.Clients[message.Sender]; ok {
					client.ChatIDs[message.ChatID] = true
				}
				h.Logger.Info("User joined chat", "userID", message.Sender, "chatID", message.ChatID)

			case "message":
				h.Logger.Info("Broadcasting message",
					"chatID", message.ChatID,
					"sender", message.Sender,
					"content", message.Content)

				if users, ok := h.ChatRooms[message.ChatID]; ok {
					for userID := range users {
						if client, exists := h.Clients[userID]; exists {
							select {
							case client.Send <- mustMarshal(message):
								h.Logger.Debug("Message sent to user", "userID", userID)
							default:
								h.Logger.Warn("Failed to send message to user", "userID", userID)
								close(client.Send)
								delete(h.Clients, userID)
							}
						}
					}
				} else {
					h.Logger.Warn("Chat room not found", "chatID", message.ChatID)
				}
			}
			h.Mutex.RUnlock()
		}
	}
}

func mustMarshal(msg Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.Logger.Error("Websocket error", "error", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Hub.Logger.Error("Failed to parse message", "error", err)
			continue
		}

		msg.Sender = c.UserID

		switch msg.Type {
		case "join_chat":
			c.Hub.Logger.Info("Join chat request", "userID", c.UserID, "chatID", msg.ChatID)
		case "message":
			c.Hub.Logger.Info("New message", "userID", c.UserID, "chatID", msg.ChatID, "content", msg.Content)
		}

		c.Hub.Broadcast <- msg
	}
}

func (h *Hub) BroadcastToUser(userID string, message map[string]interface{}) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		h.Logger.Error("Failed to marshal message", "error", err)
		return
	}

	if client, exists := h.Clients[userID]; exists {
		select {
		case client.Send <- data:
			h.Logger.Debug("Message sent to user", "userID", userID, "type", message["type"])
		default:
			h.Logger.Warn("Client channel full, closing connection", "userID", userID)
			close(client.Send)
			delete(h.Clients, userID)
		}
	} else {
		h.Logger.Debug("User not connected", "userID", userID)
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}
