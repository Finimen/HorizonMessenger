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

// ChatHandler represents the chat handler
// @Summary Create a chat
// @Tags chats
// @Description Creates a new chat with the specified participants
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateChatRequest true "Data for creating a chat"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /chats [post]
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

// @Summary Get the user's chats
// @Tags chats
// @Description Returns a list of chats the user is participating in
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /chats [get]
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

// @Summary Get chat messages
// @Tags chats
// @Description Returns messages for the specified chat with pagination
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param chatId path int true "Chat ID"
// @Param limit query int false "Message limit (max 100)"
// @Param offset query int false "Offset"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /chats/{chatId}/messages [get]
func (h *ChatHandler) GetChatMessages(c *gin.Context) {
	chatIDStr := c.Param("chatId")
	if chatIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID is required"})
		return
	}

	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID is not int"})
		return
	}

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

// @Summary Delete chat
// @Tags chats
// @Description Deletes a chat (chat members only)
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param chatId path int true "Chat ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /chats/{chatId} [delete]
func (h *ChatHandler) DeleteChat(c *gin.Context) {
	chatIdStr := c.Param("chatId")
	if chatIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID is required"})
		return
	}

	chatID, err := strconv.Atoi(chatIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID is not int"})
		return
	}

	username := c.GetString("username")

	if chatIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	err = h.service.DeleteChat(c.Request.Context(), chatID, username)

	if err != nil {
		h.logger.Error("Failed to delete chat", "error", err, "chatID", chatID, "userID", username)

		switch err {
		case services.ErrChatNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		case services.ErrNotChatMember:
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this chat"})
		case services.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		}
		return
	}

	h.logger.Info("Chat deleted successfully", "chatID", chatID, "userID", username)
	c.JSON(http.StatusOK, gin.H{"message": "Chat deleted successfully"})
}
