package services

import (
	"context"
	"errors"
	"log/slog"
	"massager/internal/models"
	"massager/internal/ports"
	websocket "massager/internal/websocet"
	"time"
)

type ChatService struct {
	chatRepo    ports.IChatRepository
	messageRepo ports.IMessageRepository
	userRepo    ports.IUserRepository
	chatStore   *MemoryChatStore
	logger      *slog.Logger
	wsHub       *websocket.Hub
}

func NewChatService(chatRepo ports.IChatRepository, messageRepo ports.IMessageRepository, userRepo ports.IUserRepository, logger *slog.Logger) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		messageRepo: messageRepo,
		userRepo:    userRepo,
		chatStore:   NewMemoryChatStore(chatRepo),
		logger:      logger,
	}
}

func (s *ChatService) SetWSHub(wsHub *websocket.Hub) {
	s.wsHub = wsHub
}

func (s *ChatService) notifyChatCreated(chat *models.Chat, createdBy string) {
	if s.wsHub == nil {
		return
	}

	notification := map[string]interface{}{
		"type":       "chat_created",
		"chat_id":    chat.ID,
		"chat_name":  chat.Name,
		"members":    chat.Members,
		"created_by": createdBy,
	}

	for _, member := range chat.Members {
		if member != createdBy {
			s.wsHub.BroadcastToUser(member, notification)
		}
	}

	s.logger.Info("notified chat members", "chatID", chat.ID, "members", chat.Members)
}

func (s *ChatService) CreateChat(ctx context.Context, chatName string, memberIDs []string) (int, error) {
	if chatName == "" {
		return 0, ErrInvalidInput
	}

	if len(memberIDs) < 2 {
		return 0, ErrInsufficientMembers
	}

	for _, userID := range memberIDs {
		user, err := s.userRepo.GetUserByName(ctx, userID)
		if err != nil {
			s.logger.Error("failed to check user existence", "userID", userID, "error", err)
			return 0, ErrUserNotFound
		}
		if user == nil {
			s.logger.Warn("user not found", "userID", userID)
			return 0, ErrUserNotFound
		}
	}

	chatID, err := s.chatRepo.CreateChat(ctx, chatName, memberIDs)
	if err != nil {
		s.logger.Error("failed to create chat in repository", "error", err)
		return 0, err
	}

	chat := &models.Chat{
		ID:        chatID,
		Name:      chatName,
		Members:   memberIDs,
		CreatedAt: time.Now(),
	}

	var createdBy string
	if len(memberIDs) > 0 {
		createdBy = memberIDs[0]
	}

	s.notifyChatCreated(chat, createdBy)

	s.logger.Info("chat created successfully", "chatID", chatID, "chatName", chatName, "memberCount", len(memberIDs))
	return chatID, nil
}

func (s *ChatService) GetUserChats(ctx context.Context, userID string) ([]models.Chat, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.userRepo.GetUserByName(ctx, userID)
	if err != nil {
		s.logger.Error("failed to check user existence", "userID", userID, "error", err)
		return nil, ErrUserNotFound
	}
	if user == nil {
		s.logger.Warn("user not found", "userID", userID)
		return nil, ErrUserNotFound
	}

	chatPointers := *s.chatStore.GetUserChats(userID)

	chats := make([]models.Chat, len(chatPointers))
	for i, chatPtr := range chatPointers {
		chats[i] = chatPtr
	}

	s.logger.Info("retrieved user chats", "userID", userID, "chatCount", len(chats))
	return chats, nil
}

func (s *ChatService) SendMessage(ctx context.Context, senderID, content string, chatID int) error {
	s.logger.Info("SendMessage called", "chatID", chatID, "senderID", senderID, "content", content)
	if senderID == "" || content == "" {
		return ErrInvalidInput
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		s.logger.Error("failed to check chat existence", "chatID", chatID, "error", err)
		return err
	}
	if chat == nil {
		s.logger.Warn("chat not found", "chatID", chatID)
		return ErrChatNotFound
	}

	isMember := false
	for _, member := range chat.Members {
		if member == senderID {
			isMember = true
			break
		}
	}

	if !isMember {
		s.logger.Warn("user is not a member of the chat", "userID", senderID, "chatID", chatID)
		return ErrNotChatMember
	}

	err = s.messageRepo.CreateMessage(ctx, senderID, content, chatID)
	s.logger.Info("chat created successfully", "chatID", chatID, "senderID", senderID)
	if err != nil {
		s.logger.Error("failed to send message", "chatID", chatID, "senderID", senderID, "error", err)
		return err
	}

	s.logger.Info("message sent successfully", "chatID", chatID, "senderID", senderID)
	return nil
}

func (s *ChatService) GetChatMessages(ctx context.Context, chatID, limit, offset int) ([]models.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	messages, err := s.messageRepo.GetMessages(ctx, chatID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get chat messages", "chatID", chatID, "error", err)
		return nil, err
	}

	s.logger.Debug("retrieved chat messages", "chatID", chatID, "messageCount", len(messages))
	return messages, nil
}

func (s *ChatService) DeleteChat(ctx context.Context, chatID int, username string) error {
	if username == "" {
		return ErrInvalidInput
	}

	chat, err := s.chatRepo.GetChatByID(ctx, chatID)
	if err != nil {
		s.logger.Error("failed to check chat existence", "chatID", chatID, "error", err)
		return err
	}
	if chat == nil {
		s.logger.Warn("chat not found", "chatID", chatID)
		return ErrChatNotFound
	}

	isMember := false
	for _, member := range chat.Members {
		if member == username {
			isMember = true
			break
		}
	}

	if !isMember {
		s.logger.Warn("user is not a member of the chat", "userID", username, "chatID", chatID)
		return ErrNotChatMember
	}

	err = s.chatRepo.DeleteChat(ctx, chatID)
	if err != nil {
		s.logger.Error("failed to delete chat", "chatID", chatID, "error", err)
		return err
	}

	err = s.messageRepo.DeleteMessagesByChatID(ctx, chatID)
	if err != nil {
		s.logger.Error("failed to delete chat messages", "chatID", chatID, "error", err)
	}

	s.notifyChatDeleted(chatID, chat.Members, username)

	s.logger.Info("chat deleted successfully", "chatID", chatID, "deletedBy", username)
	return nil
}

func (s *ChatService) notifyChatDeleted(chatID int, members []string, deletedBy string) {
	if s.wsHub == nil {
		return
	}

	notification := map[string]interface{}{
		"type":       "chat_deleted",
		"chat_id":    chatID,
		"deleted_by": deletedBy,
		"deleted_at": time.Now().Format(time.RFC3339),
	}

	for _, member := range members {
		s.wsHub.BroadcastToUser(member, notification)
	}

	s.logger.Info("notified chat members about deletion", "chatID", chatID, "members", members)
}

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrInsufficientMembers = errors.New("chat must have at least 2 members")
	ErrUserNotFound        = errors.New("user not found")
	ErrChatNotFound        = errors.New("chat not found")
	ErrNotChatMember       = errors.New("user is not a member of this chat")
)
