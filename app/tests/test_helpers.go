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
