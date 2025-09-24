package handlers

import (
	"log/slog"
	"massager/internal/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *services.AuthService
	logger  *slog.Logger
}

func NewAuthHandler(service *services.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{service: service, logger: logger}
}

func (a *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		a.logger.Warn("invalid input format", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
		return
	}

	token, err := a.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		a.logger.Warn("login failed", "username", req.Username, "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	a.logger.Info("login successful", "username", req.Username)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (a *AuthHandler) Logout(c *gin.Context) {

}

func (a *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		a.logger.Warn("invalid input format", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
		return
	}

	err := a.service.Register(c.Request.Context(), req.Username, req.Password, req.Email)
	if err != nil {
		a.logger.Warn("register failed", "username", req.Username, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"massage": "User registered successfully"})
}

func (s *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Извлечение токена из заголовка (HTTP логика)
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			s.logger.Warn("missing authorization header")
			c.JSON(401, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// 2. Очистка токена (HTTP логика)
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		// 3. Валидация токена (бизнес-логика)
		username, err := s.service.ValidateToken(c.Request.Context(), tokenStr)
		if err != nil {
			s.logger.Warn("token validation failed", "error", err)
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// 4. Установка данных в контекст (HTTP логика)
		c.Set("username", username)
		c.Set("token", tokenStr)

		s.logger.Debug("request authorized", "username", username)
		c.Next()
	}
}
