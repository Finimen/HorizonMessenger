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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type AuthService struct {
	userRepo     ports.IUserRepository
	hasher       ports.IHasher
	logger       *slog.Logger
	tokenRepo    ports.TokenRepository
	jwtKey       []byte
	emailService ports.IEmailService
	tracer       trace.Tracer
}

func NewAuthService(repo ports.IUserRepository, emailService ports.IEmailService, hasher ports.IHasher, tokenRepo ports.TokenRepository, jwtKey []byte,
	logger *slog.Logger, tracer trace.Tracer) *AuthService {
	return &AuthService{userRepo: repo, emailService: emailService, hasher: hasher, tokenRepo: tokenRepo, jwtKey: jwtKey, logger: logger, tracer: tracer}
}

func (s *AuthService) Register(c context.Context, username, password, email string) error {
	c, span := s.tracer.Start(c, "AuthService.Register")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.username", username),
		attribute.String("user.email", email),
	)
	if username == "" || password == "" || email == "" {
		s.logger.Warn("missing required fields in registration")
		span.RecordError(errors.New("missing required fields"))
		return errors.New("username, password and email are required")
	}

	s.logger.Debug("attempting user registration", "username", username, "email", email)

	existingUser, err := s.userRepo.GetUserByName(c, username)
	if err != nil {
		span.SetStatus(codes.Error, "user check failed")
		return err
	}

	if existingUser != nil {
		span.RecordError(errors.New("username already exists"))
		s.logger.Warn("username already exists", "username", username)
		return errors.New("username already exists")
	}

	verifyToken, err := generateVerificationToken()
	if err != nil {
		span.RecordError(err)
		s.logger.Error("failed to generate verification token", "error", err)
		return errors.New("registration failed")
	}

	hashedPassword, err := s.hasher.GenerateFromPassword([]byte(password), s.hasher.DefaultCost())
	if err != nil {
		span.RecordError(err)
		s.logger.Error("password hashing failed", "error", err)
		return errors.New("registration failed")
	}

	err = s.userRepo.CreateUser(c, username, string(hashedPassword), email, verifyToken)
	if err != nil {
		span.RecordError(err)
		s.logger.Warn("user creation failed", "error", err)
		return errors.New("registration failed")
	}

	if err := s.emailService.SendVerificationEmail(email, verifyToken); err != nil {
		span.RecordError(err)
		s.logger.Warn("failed to send verification email", "error", err)
	}

	span.SetStatus(codes.Ok, "registration successful")
	s.logger.Info("user registered successfully", "username", username)
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	ctx, span := s.tracer.Start(ctx, "AuthService.VerifyEmail")
	defer span.End()

	span.SetAttributes(attribute.String("verification.token", token))

	if token == "" {
		span.RecordError(errors.New("verification token is required"))
		return errors.New("verifiaction token is requered")
	}

	user, err := s.userRepo.GetUserByVerifyToken(ctx, token)
	if err != nil {
		span.RecordError(err)
		s.logger.Warn("failed to find user by verifiaction token", "error", err)
		return errors.New("invalid verification token")
	}

	if user.IsVerefied {
		span.RecordError(errors.New("email already verified"))
		return errors.New("email already verefied")
	}

	err = s.userRepo.MarkUserAsVerified(ctx, user.Username)
	if err != nil {
		span.RecordError(err)
		s.logger.Error("failed to mark user as verified", "username", user.Username, "error", err)
		return errors.New("verification failed")
	}

	span.SetStatus(codes.Ok, "email verified")
	s.logger.Info("email verefied successfully", "uername", user.Username)
	return nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.VerifyEmail")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.username", username),
	)

	if username == "" || password == "" {
		err := errors.New("username and password are required")
		span.RecordError(err)
		s.logger.Warn("empty username or password")
		return "", err
	}

	s.logger.Debug("attempting login", "username", username)

	user, err := s.userRepo.GetUserByName(ctx, username)
	if err != nil {
		span.RecordError(err)
		s.logger.Warn("user not found", "username", username, "error", err)
		return "", errors.New("invalid credentials")
	}

	if user == nil {
		span.RecordError(errors.New("user not found"))
		s.logger.Warn("user not found", "username", username)
		return "", errors.New("invalid credentials")
	}

	if !user.IsVerefied {
		span.RecordError(errors.New("email not verified"))
		s.logger.Warn("attempt to login with unverified email", "username", username)
		return "", errors.New("email not verified. Please check your email for verification link")
	}

	if err := s.hasher.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		span.RecordError(err)
		s.logger.Warn("invalid password", "username", username)
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		span.RecordError(err)
		s.logger.Error("token generation failed", "error", err)
		return "", errors.New("authentication failed")
	}

	span.SetStatus(codes.Ok, "login successful")
	s.logger.Info("login successful", "username", username)
	return tokenString, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "AuthService.ValidateToken")
	defer span.End()

	if tokenString == "" {
		err := errors.New("token is required")
		span.RecordError(err)
		return "", err
	}

	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])
	span.SetAttributes(attribute.String("token.hash", tokenHash))

	isRevoked, err := s.tokenRepo.IsRevoked(ctx, tokenHash)
	if err != nil {
		span.RecordError(err)
		s.logger.Error("token revocation check failed", "error", err)
		return "", err
	}
	if isRevoked {
		var err = errors.New("token revoked")
		span.RecordError(err)
		return "", err
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtKey, nil
	})

	if err != nil {
		span.RecordError(err)
		s.logger.Warn("token parsing failed", "error", err)
		return "", errors.New("invalid token")
	}

	if !token.Valid {
		err := errors.New("invalid token")
		span.RecordError(err)
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err := errors.New("invalid token claims")
		span.RecordError(err)
		return "", err
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		err := errors.New("token expiration missing")
		span.RecordError(err)
		return "", err
	}

	if time.Now().Unix() > int64(exp) {
		err := errors.New("token expired")
		span.RecordError(err)
		return "", err
	}

	username, ok := claims["username"].(string)
	if !ok || username == "" {
		err := errors.New("username missing in token")
		span.RecordError(err)
		return "", err
	}
	span.SetAttributes(attribute.String("token.username", username))
	span.SetStatus(codes.Ok, "token valid")
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
