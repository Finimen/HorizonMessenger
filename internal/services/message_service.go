package services

import (
	"context"
	"massager/internal/repositories"
)

type MessageService struct {
	messageRepo repositories.MessageRepository
}

func NewMessageService(messageRepo repositories.MessageRepository) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
	}
}

func (s *MessageService) SendMessage(ctx context.Context, fromUserID, toUserID, content string) error {
	return s.messageRepo.Create(ctx, fromUserID, toUserID, content)
}

func (s *MessageService) GetMessages(ctx context.Context, userID, otherUserID string, limit, offset int) ([]repositories.Message, error) {
	return s.messageRepo.GetConversation(ctx, userID, otherUserID, limit, offset)
}
