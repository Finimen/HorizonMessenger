package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"massager/app/config"
	"massager/internal/models"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type MockRepository struct {
	mock.Mock
}

type MockHasher struct {
	mock.Mock
}

type MockEmailService struct {
	mock.Mock
}

func NoopTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test-tracer")
}

type MockChatRepository struct {
	mock.Mock
}

func (m *MockChatRepository) CreateChat(ctx context.Context, name string, members []string) (int, error) {
	args := m.Called(ctx, name, members)
	return args.Int(0), args.Error(1)
}

func (m *MockChatRepository) GetUserChats(ctx context.Context, userID string) (*[]models.Chat, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*[]models.Chat), args.Error(1)
}

func (m *MockChatRepository) GetChatByID(ctx context.Context, chatID int) (*models.Chat, error) {
	args := m.Called(ctx, chatID)
	return args.Get(0).(*models.Chat), args.Error(1)
}

func (m *MockChatRepository) DeleteChat(ctx context.Context, chatID int) error {
	args := m.Called(ctx, chatID)
	return args.Error(0)
}

type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) CreateMessage(ctx context.Context, senderID, content string, chatID int) error {
	args := m.Called(ctx, senderID, content, chatID)
	return args.Error(0)
}

func (m *MockMessageRepository) GetMessages(ctx context.Context, chatID, limit, offset int) ([]models.Message, error) {
	args := m.Called(ctx, chatID, limit, offset)
	return args.Get(0).([]models.Message), args.Error(1)
}

func (m *MockMessageRepository) DeleteMessagesByChatID(ctx context.Context, chatID int) error {
	args := m.Called(ctx, chatID)
	return args.Error(0)
}

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) GetUserByName(ctx context.Context, name string) (*models.User, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockEmailService) NewEmailService(config config.EmailConfig, loggger *slog.Logger) *MockEmailService {
	args := m.Called(config, loggger)
	return args.Get(0).(*MockEmailService)
}

func (m *MockEmailService) SendVerificationEmail(email, token string) error {
	args := m.Called(email, token)
	return args.Error(0)
}

func (m *MockHasher) GenerateFromPassword(password []byte, cost int) ([]byte, error) {
	args := m.Called(password, cost)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockHasher) CompareHashAndPassword(storedPaswsord []byte, userPassword []byte) error {
	args := m.Called(storedPaswsord, userPassword)
	return args.Error(0)
}

func (m *MockHasher) DefaultCost() int {
	return m.Called().Int(0)
}

func (m *MockRepository) GetUserByName(ctx context.Context, name string) (*models.User, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) GetUserByVerifyToken(ctx context.Context, name string) (*models.User, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockRepository) CreateUser(ctx context.Context, name, email, password, token string) error {
	args := m.Called(ctx, name, email, password, token)
	return args.Error(0)
}

func (m *MockRepository) MarkUserAsVerified(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func CreateTestRequest(url, method string, body interface{}) *http.Request {
	var buffer bytes.Buffer
	if body != nil {
		json.NewEncoder(&buffer).Encode(body)
	}

	req := httptest.NewRequest(method, url, &buffer)
	req.Header.Set("Content-Type", "application/json")

	return req
}

func ExecuteHandler(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}
