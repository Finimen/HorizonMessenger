package websocket

// PROPRIETARY AND CONFIDENTIAL
// This code contains trade secrets and confidential material of Finimen Sniper / FSC.
// Any unauthorized use, disclosure, or duplication is strictly prohibited.
// © 2025 Finimen Sniper / FSC. All rights reserved.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"massager/internal/models"
	"massager/internal/ports"
	"massager/internal/services/keying"

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
	ChatIDs map[int]bool
}

type Hub struct {
	Clients     map[string]*Client
	ChatRooms   map[int]map[string]bool
	Broadcast   chan models.Message
	Register    chan *Client
	Unregister  chan *Client
	Mutex       sync.RWMutex
	ChatService ports.IMessageService
	Logger      *slog.Logger
}

func NewHub(chatService ports.IMessageService, logger *slog.Logger) *Hub {
	return &Hub{
		Clients:     make(map[string]*Client),
		ChatRooms:   make(map[int]map[string]bool),
		Broadcast:   make(chan models.Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		ChatService: chatService,
		Logger:      logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			h.Clients[client.UserID] = client
			if client.ChatIDs == nil {
				client.ChatIDs = make(map[int]bool)
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
				var err error
				message.Content, err = keying.Decrypt(message.Key, message.Content)

				if err != nil {
					log.Fatal(err)
				}

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

func mustMarshal(msg models.Message) []byte {
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

		var rawMsg map[string]interface{}
		if err := json.Unmarshal(message, &rawMsg); err != nil {
			c.Hub.Logger.Error("Failed to parse message", "error", err)
			continue
		}

		var chatID int
		if rawChatID, ok := rawMsg["chat_id"]; ok {
			switch v := rawChatID.(type) {
			case float64:
				chatID = int(v)
			case int:
				chatID = v
			case string:
				if parsed, err := strconv.Atoi(v); err == nil {
					chatID = parsed
				} else {
					c.Hub.Logger.Error("Invalid chat_id format", "chat_id", v)

					errorMsg := map[string]interface{}{
						"type":    "error",
						"error":   "Invalid chat ID format",
						"chat_id": v,
					}
					errorData, _ := json.Marshal(errorMsg)
					c.Send <- errorData
					continue
				}
			default:
				c.Hub.Logger.Error("Unknown chat_id type", "type", fmt.Sprintf("%T", v))
				continue
			}
		}

		msgType, _ := rawMsg["type"].(string)

		c.Hub.Logger.Info("Processing message",
			"type", msgType,
			"chatID", chatID,
			"sender", c.UserID)

		if msgType == "message" {
			content, _ := rawMsg["content"].(string)

			err := c.Hub.ChatService.SendMessage(context.Background(), c.UserID, content, chatID)
			if err != nil {
				c.Hub.Logger.Error("Failed to send message",
					"error", err,
					"userID", c.UserID,
					"chatID", chatID)

				errorMsg := map[string]interface{}{
					"type":    "error",
					"error":   err.Error(),
					"chat_id": chatID,
					"details": "You are not a member of this chat or chat doesn't exist",
				}
				errorData, _ := json.Marshal(errorMsg)
				c.Send <- errorData
				continue
			}

			key, _ := keying.GenerateKeyAES128()
			encryptedContent, _ := keying.Encrypt(key, content)

			msg := models.Message{
				Type:    "message",
				ChatID:  chatID,
				Sender:  c.UserID,
				Content: encryptedContent,
				Key:     key,
			}

			c.Hub.Broadcast <- msg
		} else if msgType == "join_chat" {
			msg := models.Message{
				Type:   "join_chat",
				ChatID: chatID,
				Sender: c.UserID,
			}
			c.Hub.Broadcast <- msg
		}
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
