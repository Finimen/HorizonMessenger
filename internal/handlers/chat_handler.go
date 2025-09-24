package handlers

import (
	"log/slog"
	"massager/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	service *services.ChatService
	logger  *slog.Logger
}

func NewChatHandler(service *services.ChatService, logger *slog.Logger) *ChatHandler {
	return &ChatHandler{service: service, logger: logger}
}

func (h *ChatHandler) CreateChat(c *gin.Context) {
	var req struct {
		MemberIDs []string `json:"member_ids"`
		ChatName  string   `json:"chat_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userID := c.GetString("username")
	req.MemberIDs = append(req.MemberIDs, userID)

	chatID, err := h.service.CreateChat(c.Request.Context(), req.ChatName, req.MemberIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chat_id": chatID})
}

func (h *ChatHandler) GetUserChats(c *gin.Context) {
	userID := c.GetString("username")
	h.logger.Info("GetUserChats called", "userID", userID)

	chats, err := h.service.GetUserChats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user chats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Returning user chats", "count", len(chats))
	c.JSON(http.StatusOK, gin.H{"chats": chats})
}
