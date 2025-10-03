package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"massager/internal/ports"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService struct {
	userRepo     ports.IUserRepository
	hasher       ports.IHasher
	logger       *slog.Logger
	tokenRepo    ports.TokenRepository
	jwtKey       []byte
	emailService ports.IEmailService
}

func NewAuthService(repo ports.IUserRepository, emailService ports.IEmailService, hasher ports.IHasher, tokenRepo ports.TokenRepository, jwtKey []byte, logger *slog.Logger) *AuthService {
	return &AuthService{userRepo: repo, emailService: emailService, hasher: hasher, tokenRepo: tokenRepo, jwtKey: jwtKey, logger: logger}
}

func (s *AuthService) Register(c context.Context, username, password, email string) error {
	if username == "" || password == "" || email == "" {
		s.logger.Warn("missing required fields in registration")
		return errors.New("username, password and email are required")
	}

	s.logger.Debug("attempting user registration", "username", username, "email", email)

	existingUser, err := s.userRepo.GetUserByName(c, username)
	if err != nil {
		return err
	}

	if existingUser != nil {
		s.logger.Warn("username already exists", "username", username)
		return errors.New("username already exists")
	}

	verifyToken, err := generateVerificationToken()
	if err != nil {
		s.logger.Error("failed to generate verification token", "error", err)
		return errors.New("registration failed")
	}

	hashedPassword, err := s.hasher.GenerateFromPassword([]byte(password), s.hasher.DefaultCost())
	if err != nil {
		s.logger.Error("password hashing failed", "error", err)
		return errors.New("registration failed")
	}

	err = s.userRepo.CreateUser(c, username, string(hashedPassword), email, verifyToken)
	if err != nil {
		s.logger.Warn("user creation failed", "error", err)
		return errors.New("registration failed")
	}

	if err := s.emailService.SendVerificationEmail(email, verifyToken); err != nil {
		s.logger.Warn("failed to send verification email", "error", err)
	}

	s.logger.Info("user registered successfully", "username", username)
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("verifiaction token is requered")
	}

	user, err := s.userRepo.GetUserByVerifyToken(ctx, token)
	if err != nil {
		s.logger.Warn("failed to find user by verifiaction token", "error", err)
		return errors.New("invalid verification token")
	}

	if user.IsVerefied {
		return errors.New("email already verefied")
	}

	err = s.userRepo.MarkUserAsVerified(ctx, user.Username)
	if err != nil {
		s.logger.Error("failed to mark user as verified", "username", user.Username, "error", err)
		return errors.New("verification failed")
	}

	s.logger.Info("email verefied successfully", "uername", user.Username)
	return nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	if username == "" || password == "" {
		s.logger.Warn("empty username or password")
		return "", errors.New("username and password are required")
	}

	s.logger.Debug("attempting login", "username", username)

	user, err := s.userRepo.GetUserByName(ctx, username)
	if err != nil {
		s.logger.Warn("user not found", "username", username, "error", err)
		return "", errors.New("invalid credentials")
	}

	if user == nil {
		s.logger.Warn("user not found", "username", username)
		return "", errors.New("invalid credentials")
	}

	if !user.IsVerefied {
		s.logger.Warn("attempt to login with unverified email", "username", username)
		return "", errors.New("email not verified. Please check your email for verification link")
	}

	if err := s.hasher.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.logger.Warn("invalid password", "username", username)
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		s.logger.Error("token generation failed", "error", err)
		return "", errors.New("authentication failed")
	}

	s.logger.Info("login successful", "username", username)
	return tokenString, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("token is required")
	}

	// Проверка отозванных токенов
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	isRevoked, err := s.tokenRepo.IsRevoked(ctx, tokenHash)
	if err != nil {
		s.logger.Error("token revocation check failed", "error", err)
		return "", err
	}
	if isRevoked {
		return "", errors.New("token revoked")
	}

	// Парсинг токена
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtKey, nil
	})

	if err != nil {
		s.logger.Warn("token parsing failed", "error", err)
		return "", errors.New("invalid token")
	}

	if !token.Valid {
		return "", errors.New("invalid token")
	}

	// Извлечение claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	// Проверка expiration
	exp, ok := claims["exp"].(float64)
	if !ok {
		return "", errors.New("token expiration missing")
	}

	if time.Now().Unix() > int64(exp) {
		return "", errors.New("token expired")
	}

	username, ok := claims["username"].(string)
	if !ok || username == "" {
		return "", errors.New("username missing in token")
	}

	s.logger.Debug("token validated", "username", username)
	return username, nil
}

func (s *AuthService) RevokeToken(ctx context.Context, tokenString string, expiration time.Duration) error {
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])
	return s.tokenRepo.Revoke(ctx, tokenHash, expiration)
}

func (s *AuthService) GetVerificationToken(ctx context.Context, username string) (string, error) {
	user, err := s.userRepo.GetUserByName(ctx, username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("user not found")
	}
	return user.VerifyToken, nil
}

func (s *AuthService) GetUserVerificationStatus(ctx context.Context, username string) (bool, error) {
	user, err := s.userRepo.GetUserByName(ctx, username)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, errors.New("user not found")
	}
	return user.IsVerefied, nil
}
