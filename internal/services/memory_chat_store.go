package services

import (
	"massager/internal/models"
	"sync"
)

type MemoryChatStore struct {
	chats map[string]*models.Chat
	mu    sync.RWMutex
}

func NewMemoryChatStore() *MemoryChatStore {
	return &MemoryChatStore{
		chats: make(map[string]*models.Chat),
	}
}

func (s *MemoryChatStore) CreateChat(chat *models.Chat) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chats[chat.ID] = chat
}

func (s *MemoryChatStore) GetChatByID(chatID string) *models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chats[chatID]
}

func (s *MemoryChatStore) GetUserChats(userID string) []*models.Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var userChats []*models.Chat
	for _, chat := range s.chats {
		for _, member := range chat.Members {
			if member == userID {
				userChats = append(userChats, chat)
				break
			}
		}
	}
	return userChats
}
