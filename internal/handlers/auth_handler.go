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

// @Summary User login
// @Tags auth
// @Description Authenticates the user and returns a token
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Data for login"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
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

// @Summary User logout
// @Tags auth
// @Description Terminates the user session
// @Accept json
// @Produce json
// @Success 200
// @Router /auth/logout [post]
func (a *AuthHandler) Logout(c *gin.Context) {

}

// AuthHandler represents the authentication handler
// @Summary User registration
// @Tags auth
// @Description Creates a new user in the system
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Data for registration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /auth/register [post]
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

// @Summary Authentication middleware
// @Tags auth
// @Description Checks the JWT token in the Authorization header
// @Security BearerAuth
func (s *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			s.logger.Warn("missing authorization header")
			c.JSON(401, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

		username, err := s.service.ValidateToken(c.Request.Context(), tokenStr)
		if err != nil {
			s.logger.Warn("token validation failed", "error", err)
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("username", username)
		c.Set("token", tokenStr)

		s.logger.Debug("request authorized", "username", username)
		c.Next()
	}
}
