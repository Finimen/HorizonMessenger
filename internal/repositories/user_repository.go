package repositories

import (
	"context"
	"database/sql"
	"massager/internal/models"

	_ "embed"
)

//go:embed migrations/001_create_user_table_up.sql
var createUserTableQuery string

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) (*UserRepository, error) {
	var repo = UserRepository{db: db}
	var _, err = repo.db.Exec(string(createUserTableQuery))
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *UserRepository) GetUserByName(ctx context.Context, name string) (*models.User, error) {
	var password, email string
	query := "SELECT passwordHash, email FROM users WHERE username = ?"
	row := r.db.QueryRowContext(ctx, query, name)
	err := row.Scan(&password, &email)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return models.NewUser(name, password, email), err
}

func (r *UserRepository) CreateUser(ctx context.Context, username, hashedPassword, email string) error {
	user := models.User{
		Username: username,
		Password: hashedPassword,
		Email:    email,
	}

	_, err := r.db.ExecContext(ctx, "INSERT INTO users (username, passwordHash, email) VALUES (?, ?, ?)", user.Username, user.Password, user.Email)

	return err
}
