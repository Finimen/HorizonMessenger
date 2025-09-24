package handlers

import (
	"log/slog"
	"massager/internal/services"
	internalWebsocket "massager/internal/websocet"
	"net/http"

	libWebsocket "github.com/gorilla/websocket"

	"github.com/gin-gonic/gin"
)

type WebsocetHandler struct {
	Hub         *internalWebsocket.Hub
	AuthService *services.AuthService
	Logger      *slog.Logger
}

func NewWebSocketHandler(hub *internalWebsocket.Hub, authService *services.AuthService, logger *slog.Logger) *WebsocetHandler {
	return &WebsocetHandler{
		Hub:         hub,
		AuthService: authService,
		Logger:      logger,
	}
}

func (h *WebsocetHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		cookie, err := c.Request.Cookie("token")
		if err == nil {
			token = cookie.Value
		}
	}

	userID, err := h.AuthService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		h.Logger.Warn("Unauthorized WebSocket connection attempt")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	upgrader := libWebsocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.Logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	client := &internalWebsocket.Client{
		Hub:    h.Hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	client.Hub.Register <- client

	go client.WritePump()
	go client.ReadPump()

	h.Logger.Info("WebSocket connection established", "userID", userID)
}
