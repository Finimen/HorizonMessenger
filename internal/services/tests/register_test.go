package services_test

import (
	"errors"
	"log/slog"
	"massager/app/tests"
	"massager/internal/handlers"
	"massager/internal/models"
	"massager/internal/ports"
	"massager/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_TableDrive(t *testing.T) {
	var ts = []struct {
		name         string
		requestBody  map[string]interface{}
		setupMocks   func(*tests.MockRepository, *tests.MockHasher, *tests.MockEmailService)
		expectedCode int
		expectedBody string
	}{
		{
			name: "Successful registration",
			requestBody: map[string]interface{}{
				"username": "validuser",
				"email":    "validemail@gmail.com",
				"password": "validpassword",
			},
			setupMocks: func(mr *tests.MockRepository, mh *tests.MockHasher, mes *tests.MockEmailService) {
				// Mock: User doesn't exist
				mr.On("GetUserByName", mock.Anything, "validuser").Return((*models.User)(nil), nil)

				// Mock: Password hashing
				mh.On("GenerateFromPassword", []byte("validpassword"), bcrypt.DefaultCost).Return([]byte("hashed_password"), nil)
				mh.On("DefaultCost").Return(bcrypt.DefaultCost)

				// Mock: User creation
				mr.On("CreateUser", mock.Anything, "validuser", "hashed_password", "validemail@gmail.com", mock.AnythingOfType("string")).Return(nil)

				// Mock: Email sending
				mes.On("SendVerificationEmail", "validemail@gmail.com", mock.AnythingOfType("string")).Return(nil)
			},
			expectedCode: http.StatusOK,
			expectedBody: "User registered successfully",
		},
		{
			name: "empty username",
			requestBody: map[string]interface{}{
				"username": "",
				"password": "password123",
				"email":    "test@gmail.com",
			},
			setupMocks: func(mur *tests.MockRepository, mph *tests.MockHasher, mes *tests.MockEmailService) {
				// No mocks needed since validation fails early
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "username, password and email are required", // FIX: Updated expected error message
		},
		{
			name: "empty password",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"password": "",
				"email":    "test@gmail.com",
			},
			setupMocks: func(mockRepo *tests.MockRepository, mockHasher *tests.MockHasher, mes *tests.MockEmailService) {
				// No mocks needed since validation fails early
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "username, password and email are required", // FIX: Updated expected error message
		},
		{
			name: "empty email",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
				"email":    "",
			},
			setupMocks: func(mockRepo *tests.MockRepository, mockHasher *tests.MockHasher, mes *tests.MockEmailService) {
				// No mocks needed since validation fails early
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "username, password and email are required", // FIX: Updated expected error message
		},
		{
			name: "username already exists",
			requestBody: map[string]interface{}{
				"username": "existinguser",
				"password": "password123",
				"email":    "test@gmail.com",
			},
			setupMocks: func(mockRepo *tests.MockRepository, mockHasher *tests.MockHasher, mes *tests.MockEmailService) {
				// Mock: User already exists
				existingUser := &models.User{Username: "existinguser"}
				mockRepo.On("GetUserByName", mock.Anything, "existinguser").Return(existingUser, nil)
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "username already exists",
		},
		{
			name: "password hashing fails",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
				"email":    "test@gmail.com",
			},
			setupMocks: func(mockRepo *tests.MockRepository, mockHasher *tests.MockHasher, mes *tests.MockEmailService) {
				// Mock: User doesn't exist
				mockRepo.On("GetUserByName", mock.Anything, "testuser").Return((*models.User)(nil), nil)

				// Mock: Password hashing fails
				mockHasher.On("GenerateFromPassword", []byte("password123"), bcrypt.DefaultCost).Return([]byte(""), errors.New("hashing failed"))
				mockHasher.On("DefaultCost").Return(bcrypt.DefaultCost)
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "registration failed",
		},
		{
			name: "user creation fails",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
				"email":    "test@gmail.com",
			},
			setupMocks: func(mockRepo *tests.MockRepository, mockHasher *tests.MockHasher, mes *tests.MockEmailService) {
				// Mock: User doesn't exist
				mockRepo.On("GetUserByName", mock.Anything, "testuser").Return((*models.User)(nil), nil)

				// Mock: Password hashing succeeds
				mockHasher.On("GenerateFromPassword", []byte("password123"), bcrypt.DefaultCost).Return([]byte("hashed_password"), nil)
				mockHasher.On("DefaultCost").Return(bcrypt.DefaultCost)

				// Mock: User creation fails
				mockRepo.On("CreateUser", mock.Anything, "testuser", "hashed_password", "test@gmail.com", mock.AnythingOfType("string")).Return(errors.New("database error"))
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "registration failed",
		},
		{
			name: "email sending fails",
			requestBody: map[string]interface{}{
				"username": "testuser",
				"password": "password123",
				"email":    "test@gmail.com",
			},
			setupMocks: func(mockRepo *tests.MockRepository, mockHasher *tests.MockHasher, mes *tests.MockEmailService) {
				// Mock: User doesn't exist
				mockRepo.On("GetUserByName", mock.Anything, "testuser").Return((*models.User)(nil), nil)

				// Mock: Password hashing
				mockHasher.On("GenerateFromPassword", []byte("password123"), bcrypt.DefaultCost).Return([]byte("hashed_password"), nil)
				mockHasher.On("DefaultCost").Return(bcrypt.DefaultCost)

				// Mock: User creation succeeds
				mockRepo.On("CreateUser", mock.Anything, "testuser", "hashed_password", "test@gmail.com", mock.AnythingOfType("string")).Return(nil)

				// Mock: Email sending fails
				mes.On("SendVerificationEmail", "test@gmail.com", mock.AnythingOfType("string")).Return(errors.New("email error"))
			},
			expectedCode: http.StatusOK, // Registration should still succeed even if email fails
			expectedBody: "User registered successfully",
		},
	}

	for _, tt := range ts {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepository := &tests.MockRepository{}
			mockHasher := &tests.MockHasher{}
			mockEmailService := &tests.MockEmailService{}
			var tokenRepository ports.TokenRepository
			jwtKey := []byte("test_key")
			logger := slog.Default()

			tt.setupMocks(mockRepository, mockHasher, mockEmailService)

			var authService = services.NewAuthService(
				mockRepository, mockEmailService, mockHasher,
				tokenRepository, jwtKey, logger)

			var handler = handlers.NewAuthHandler(authService, logger)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = tests.CreateTestRequest("/register", http.MethodPost, tt.requestBody)

			handler.Register(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			mockRepository.AssertExpectations(t)
			mockHasher.AssertExpectations(t)
			mockEmailService.AssertExpectations(t)
		})
	}
}
