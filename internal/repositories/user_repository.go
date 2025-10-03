package repositories

import (
	"context"
	"database/sql"
	"log/slog"
	"massager/internal/models"

	_ "embed"
)

//go:embed migrations/001_create_user_table_up.sql
var createUserTableQuery string

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB, logger *slog.Logger) (*UserRepository, error) {
	var repo = UserRepository{db: db}
	var _, err = repo.db.Exec(string(createUserTableQuery))
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return &repo, nil
}

func (r *UserRepository) GetUserByName(ctx context.Context, name string) (*models.User, error) {
	var password, email string
	var isVerified bool
	var verifyToken sql.NullString

	query := "SELECT passwordHash, email, is_verified, verify_token FROM users WHERE username = $1"
	row := r.db.QueryRowContext(ctx, query, name)
	err := row.Scan(&password, &email, &isVerified, &verifyToken)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var user = models.NewUser(name, password, email)
	user.IsVerefied = isVerified
	if verifyToken.Valid {
		user.VerifyToken = verifyToken.String
	}

	return user, err
}

func (r *UserRepository) GetUserByVerifyToken(ctx context.Context, token string) (*models.User, error) {
	var username, password, email string
	var isVerified bool

	query := "SELECT username, passwordHash, email, is_verified FROM users WHERE verify_token = $1"
	row := r.db.QueryRowContext(ctx, query, token)
	err := row.Scan(&username, &password, &email, &isVerified)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	user := models.NewUser(username, password, email)
	user.IsVerefied = isVerified
	user.VerifyToken = token

	return user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, username, hashedPassword, email, verifyToken string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO users (username, passwordHash, email, verify_token) VALUES ($1, $2, $3, $4)",
		username, hashedPassword, email, verifyToken)

	return err
}

func (r *UserRepository) MarkUserAsVerified(ctx context.Context, username string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET is_verified = TRUE, verify_token = NULL WHERE username = $1",
		username)
	return err
}
