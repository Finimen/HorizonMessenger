package services_test

import (
	"context"
	"errors"
	"log/slog"
	"massager/app/tests"
	"massager/internal/models"
	"massager/internal/services"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChat_CreateChat(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	ts := []struct {
		name          string
		chatName      string
		memberIDs     []string
		setupMocks    func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository)
		expectedID    int
		expectedError error
	}{
		{
			name:      "Successful chat creation",
			chatName:  "Test Chat",
			memberIDs: []string{"user1", "user2", "user3"},
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
				userRepo.On("GetUserByName", ctx, "user1").Return(&models.User{Username: "user1"}, nil)
				userRepo.On("GetUserByName", ctx, "user2").Return(&models.User{Username: "user2"}, nil)
				userRepo.On("GetUserByName", ctx, "user3").Return(&models.User{Username: "user3"}, nil)

				chatRepo.On("CreateChat", ctx, "Test Chat", []string{"user1", "user2", "user3"}).Return(123, nil)
			},
			expectedID:    123,
			expectedError: nil,
		},
		{
			name:      "Empty chat name",
			chatName:  "",
			memberIDs: []string{"user1", "user2"},
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
			},
			expectedID:    0,
			expectedError: services.ErrInvalidInput,
		},
		{
			name:      "Not enough participants",
			chatName:  "Test Chat",
			memberIDs: []string{"user1"},
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
			},
			expectedID:    0,
			expectedError: services.ErrInsufficientMembers,
		},
		{
			name:      "User not found",
			chatName:  "Test Chat",
			memberIDs: []string{"user1", "user2"},
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
				userRepo.On("GetUserByName", ctx, "user1").Return(&models.User{Username: "user1"}, nil)
				userRepo.On("GetUserByName", ctx, "user2").Return((*models.User)(nil), nil)
			},
			expectedID:    0,
			expectedError: services.ErrUserNotFound,
		},
		{
			name:      "Error creating chat in repository",
			chatName:  "Test Chat",
			memberIDs: []string{"user1", "user2"},
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
				userRepo.On("GetUserByName", ctx, "user1").Return(&models.User{Username: "user1"}, nil)
				userRepo.On("GetUserByName", ctx, "user2").Return(&models.User{Username: "user2"}, nil)
				chatRepo.On("CreateChat", ctx, "Test Chat", []string{"user1", "user2"}).Return(0, errors.New("db error"))
			},
			expectedID:    0,
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range ts {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chatRepo := &tests.MockChatRepository{}
			userRepo := &tests.MockRepository{}
			messageRepo := &tests.MockMessageRepository{}

			tt.setupMocks(chatRepo, userRepo, messageRepo)

			service := services.NewChatService(chatRepo, messageRepo, userRepo, logger)
			chatID, err := service.CreateChat(ctx, tt.chatName, tt.memberIDs)

			assert.Equal(t, tt.expectedID, chatID)
			assert.Equal(t, tt.expectedError, err)

			chatRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			messageRepo.AssertExpectations(t)
		})
	}
}

func TestChatService_GetUserChats(t *testing.T) {
	logger := slog.Default()
	ctx := context.Background()

	ts := []struct {
		name          string
		userID        string
		setupMocks    func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository)
		expectedChats []models.Chat
		expectedError error
	}{
		{
			name:   "Successfully retrieved user chats",
			userID: "user1",
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
				userRepo.On("GetUserByName", ctx, "user1").Return(&models.User{Username: "user1"}, nil)

				expectedChats := &[]models.Chat{
					{ID: 1, Name: "Chat 1", Members: []string{"user1", "user2"}},
					{ID: 2, Name: "Chat 2", Members: []string{"user1", "user3"}},
				}
				chatRepo.On("GetUserChats", ctx, "user1").Return(expectedChats, nil)
			},
			expectedChats: []models.Chat{
				{ID: 1, Name: "Chat 1", Members: []string{"user1", "user2"}},
				{ID: 2, Name: "Chat 2", Members: []string{"user1", "user3"}},
			},
			expectedError: nil,
		},
		{
			name:   "Empty userID",
			userID: "",
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
			},
			expectedChats: nil,
			expectedError: services.ErrInvalidInput,
		},
		{
			name:   "User not found",
			userID: "unknown",
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
				userRepo.On("GetUserByName", ctx, "unknown").Return((*models.User)(nil), nil)
			},
			expectedChats: nil,
			expectedError: services.ErrUserNotFound,
		},
		{
			name:   "Repository error when retrieving chats",
			userID: "user1",
			setupMocks: func(chatRepo *tests.MockChatRepository, userRepo *tests.MockRepository, messageRepo *tests.MockMessageRepository) {
				userRepo.On("GetUserByName", ctx, "user1").Return(&models.User{Username: "user1"}, nil)
				chatRepo.On("GetUserChats", ctx, "user1").Return(&[]models.Chat{}, errors.New("db error"))
			},
			expectedChats: nil,
			expectedError: errors.New("db error"),
		},
	}

	for _, tt := range ts {
		t.Run(tt.name, func(t *testing.T) {
			chatRepo := &tests.MockChatRepository{}
			userRepo := &tests.MockRepository{}
			messageRepo := &tests.MockMessageRepository{}

			tt.setupMocks(chatRepo, userRepo, messageRepo)

			service := services.NewChatService(chatRepo, messageRepo, userRepo, logger)
			chats, err := service.GetUserChats(ctx, tt.userID)

			assert.Equal(t, tt.expectedChats, chats)
			assert.Equal(t, tt.expectedError, err)

			chatRepo.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			messageRepo.AssertExpectations(t)
		})
	}
}
