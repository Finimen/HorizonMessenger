package ports

import (
	"context"
	"massager/internal/models"
)

type IChatRepository interface {
	CreateChat(ctx context.Context, name string, members []string) (string, error)
	GetUserChats(ctx context.Context, userID string) ([]models.Chat, error)
	GetChatByID(ctx context.Context, chatID string) (*models.Chat, error)
}

type IMessageRepository interface {
	CreateMessage(ctx context.Context, chatID, senderID, content string) error
	GetMessages(ctx context.Context, chatID string, limit, offset int) ([]models.Message, error)
}
