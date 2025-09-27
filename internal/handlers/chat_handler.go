package handlers

import (
	"log/slog"
	"massager/internal/services"
	"net/http"
	"strconv"

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

func (h *ChatHandler) GetChatMessages(c *gin.Context) {
	chatIDStr := c.Param("chatId")
	if chatIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID is required"})
		return
	}

	chatID, err := strconv.Atoi(chatIDStr)

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	if limit > 100 {
		limit = 100
	}

	messages, err := h.service.GetChatMessages(c.Request.Context(), chatID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get chat messages", "error", err, "chatID", chatID)

		switch err {
		case services.ErrChatNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		case services.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		}
		return
	}

	h.logger.Info("Retrieved chat messages", "chatID", chatID, "count", len(messages))
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
