package services_test

import (
	"encoding/json"
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
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

const (
	JwtKey = "test_key"
)

func TestLogin_TableDrive(t *testing.T) {
	var ts = []struct {
		name         string
		requestBody  map[string]interface{}
		setupMocks   func(*tests.MockRepository, *tests.MockHasher)
		expectedCode int
		expectedBody string
		checkToken   bool
	}{
		{
			name: "Successful login",
			requestBody: map[string]interface{}{
				"username": "validuser",
				"password": "correctpassword",
			},
			setupMocks: func(mur *tests.MockRepository, mph *tests.MockHasher) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

				user := &models.User{
					Username:   "validuser",
					Password:   string(hashedPassword),
					IsVerefied: true,
				}
				mur.On("GetUserByName", mock.Anything, "validuser").Return(user, nil)

				mph.On("CompareHashAndPassword", []byte(user.Password), []byte("correctpassword")).Return(nil)
			},
			expectedCode: http.StatusOK,
			checkToken:   true,
		},
		{
			name: "User not found",
			requestBody: map[string]interface{}{
				"username": "nonexistent",
				"password": "password",
			},
			setupMocks: func(mur *tests.MockRepository, mph *tests.MockHasher) {
				mur.On("GetUserByName", mock.Anything, "nonexistent").Return((*models.User)(nil), nil)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "invalid credentials",
			checkToken:   false,
		},
		{
			name: "Wrong password",
			requestBody: map[string]interface{}{
				"username": "validuser",
				"password": "wrongpassword",
			},
			setupMocks: func(mur *tests.MockRepository, mph *tests.MockHasher) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
				user := &models.User{
					Username:   "validuser",
					Password:   string(hashedPassword),
					IsVerefied: true,
				}
				mur.On("GetUserByName", mock.Anything, "validuser").Return(user, nil)
				mph.On("CompareHashAndPassword", []byte(user.Password), []byte("wrongpassword")).Return(bcrypt.ErrMismatchedHashAndPassword)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "invalid credentials",
			checkToken:   false,
		},
		{
			name: "User not verified",
			requestBody: map[string]interface{}{
				"username": "unverifieduser",
				"password": "password",
			},
			setupMocks: func(mur *tests.MockRepository, mph *tests.MockHasher) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
				user := &models.User{
					Username:   "unverifieduser",
					Password:   string(hashedPassword),
					IsVerefied: false,
				}
				mur.On("GetUserByName", mock.Anything, "unverifieduser").Return(user, nil)
			},
			expectedCode: http.StatusUnauthorized,
			expectedBody: "email not verified",
			checkToken:   false,
		},
	}

	for _, tt := range ts {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRepository := &tests.MockRepository{}
			mockHasher := &tests.MockHasher{}
			var tokenRepository ports.TokenRepository
			emailService := &tests.MockEmailService{}
			jwtKey := []byte(JwtKey)
			logger := slog.Default()

			tt.setupMocks(mockRepository, mockHasher)

			var authService = services.NewAuthService(
				mockRepository, emailService, mockHasher,
				tokenRepository, jwtKey, logger)

			var handler = handlers.NewAuthHandler(authService,
				logger, tests.NoopTracer())

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request = tests.CreateTestRequest("/login", http.MethodPost, tt.requestBody)

			handler.Login(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			if tt.checkToken {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				tokenString, exists := response["token"]
				assert.True(t, exists)
				assert.NotEmpty(t, tokenString)

				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return jwtKey, nil
				})

				assert.NoError(t, err)
				assert.True(t, token.Valid)

				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					assert.Equal(t, "validuser", claims["username"])
					assert.NotEmpty(t, claims["exp"])
				}
			}

			mockRepository.AssertExpectations(t)
			mockHasher.AssertExpectations(t)
		})
	}
}
