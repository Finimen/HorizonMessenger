package handlers

// PROPRIETARY AND CONFIDENTIAL
// This code contains trade secrets and confidential material of Finimen Sniper / FSC.
// Any unauthorized use, disclosure, or duplication is strictly prohibited.
// © 2025 Finimen Sniper / FSC. All rights reserved.

import (
	"log/slog"
	"massager/internal/services"
	internalWebsocket "massager/internal/websocet"
	"net/http"

	libWebsocket "github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/trace"

	"github.com/gin-gonic/gin"
)

type WebsocetHandler struct {
	Hub         *internalWebsocket.Hub
	AuthService *services.AuthService
	Logger      *slog.Logger
	Tracer      trace.Tracer
}

func NewWebSocketHandler(hub *internalWebsocket.Hub, authService *services.AuthService, logger *slog.Logger, tracer trace.Tracer) *WebsocetHandler {
	return &WebsocetHandler{
		Hub:         hub,
		AuthService: authService,
		Logger:      logger,
		Tracer:      tracer,
	}
}

// WebsocketHandler represents a WebSocket connection handler
// @Summary WebSocket connection
// @Tags websocket
// @Description Establishes a real-time WebSocket connection
// @Param token query string false "JWT token (cookie alternative)"
// @Success 101 "Switching Protocols"
// @Failure 401 {object} map[string]string
// @Router /ws [get]
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
