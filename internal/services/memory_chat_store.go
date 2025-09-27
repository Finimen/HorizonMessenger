package services

import (
	"context"
	"log"
	"massager/internal/models"
	"massager/internal/ports"
	"sync"
)

type MemoryChatStore struct {
	chats map[int]*models.Chat
	mu    sync.RWMutex
	repo  ports.IChatRepository
}

func NewMemoryChatStore(repo ports.IChatRepository) *MemoryChatStore {
	return &MemoryChatStore{
		chats: make(map[int]*models.Chat),
		repo:  repo,
	}
}

func (s *MemoryChatStore) CreateChat(chat *models.Chat) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chats[chat.ID] = chat
	var _, err = s.repo.CreateChat(context.Background(), chat.Name, chat.Members)
	if err != nil {
		log.Fatal(err)
	}
}

func (s *MemoryChatStore) GetChatByID(chatID int) *models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chats[chatID]
}

func (s *MemoryChatStore) GetUserChats(userID string) *[]models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storedChats, err := s.repo.GetUserChats(context.Background(), userID)
	if err != nil {
		log.Printf("Error getting user chats from repo: %v", err)
		return &[]models.Chat{}
	}

	uniqueChats := make(map[int]models.Chat)

	for _, chat := range *storedChats {
		uniqueChats[chat.ID] = chat
	}

	for _, chat := range s.chats {
		for _, member := range chat.Members {
			if member == userID {
				if _, exists := uniqueChats[chat.ID]; !exists {
					uniqueChats[chat.ID] = *chat
				}
				break
			}
		}
	}

	result := make([]models.Chat, 0, len(uniqueChats))
	for _, chat := range uniqueChats {
		result = append(result, chat)
	}

	return &result
}
