package ports

import (
	"context"
	"massager/internal/models"
)

type IChatRepository interface {
	CreateChat(ctx context.Context, chatName string, memberIDs []string) (int, error)
	GetChatByID(ctx context.Context, chatID int) (*models.Chat, error)
	GetUserChats(ctx context.Context, userID string) (*[]models.Chat, error)
}

type IMessageRepository interface {
	CreateMessage(ctx context.Context, senderID, content string, chatID int) error
	GetMessages(ctx context.Context, chatID, limit, offset int) ([]models.Message, error)
}
