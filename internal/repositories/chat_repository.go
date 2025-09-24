package repositories

import (
	"context"
	"database/sql"
	"massager/internal/models"
)

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateChat(ctx context.Context, name string, members []string) (string, error) {
	// Временная реализация - создаем простой ID
	chatID := "chat_" + name + "_" + members[0]
	return chatID, nil
}

func (r *ChatRepository) GetUserChats(ctx context.Context, userID string) ([]models.Chat, error) {
	// Временная реализация - возвращаем пустой список
	return []models.Chat{}, nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, chatID string) (*models.Chat, error) {
	// Временная реализация
	return &models.Chat{
		ID:      chatID,
		Name:    "Test Chat",
		Members: []string{"user1", "user2"},
	}, nil
}
