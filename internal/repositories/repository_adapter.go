package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"massager/app/config"
	"time"

	_ "github.com/lib/pq"
)

type RepositoryAdapter struct {
	User    *UserRepository
	Chat    *ChatRepository
	Message *MessageRepository
}

func NewRepositoryAdapter(cfg config.DatabaseConfig, cfgConn config.DatabaseConnectionsConfig, logger *slog.Logger) (*RepositoryAdapter, error) {
	connection := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	var db, e = sql.Open("postgres", connection)
	if e != nil {
		return nil, e
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	logger.Info("adapter initialization: stage 1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	userRepo, err1 := NewUserRepository(db, logger)
	if err1 != nil {
		return nil, e
	}

	logger.Info("adapter initialization: stage 2")

	var chatRepo, err2 = NewChatRepository(db, logger)
	if err2 != nil {
		return nil, e
	}
	var messageRepo, err3 = NewMessageRepository(db, logger)
	if err3 != nil {
		return nil, e
	}

	logger.Info("adapter initialization: stage 3")

	return &RepositoryAdapter{User: userRepo, Message: messageRepo, Chat: chatRepo}, nil
}

func (r *RepositoryAdapter) Close(logger *slog.Logger) error {
	if err := r.User.db.Close(); err != nil {
		logger.Error("failed to close user repository", "error", err)
		return err
	}

	return nil
}

func (r *RepositoryAdapter) HealthCheck(ctx context.Context) error {
	return nil
}
