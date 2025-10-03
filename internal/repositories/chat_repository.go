package repositories

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"massager/internal/models"
	"strings"
)

//go:embed migrations/003_create_chats_table_up.sql
var createChatTableQuery string

//go:embed migrations/004_create_chat_participants_up.sql
var createСhatParticipantsQuery string

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB, logger *slog.Logger) (*ChatRepository, error) {
	var repo = ChatRepository{db: db}
	var _, err = repo.db.Exec(string(createChatTableQuery))
	_, err = repo.db.Exec(string(createСhatParticipantsQuery))
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	logger.Info("chat repository initialization stage 1")

	err = db.Ping()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	logger.Info("chat repository initialization stage 2")

	return &repo, nil
}

func (r *ChatRepository) CreateChat(ctx context.Context, name string, memberUsernames []string) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var chatId int64
	err = tx.QueryRowContext(ctx, "INSERT INTO chats (chatname) VALUES ($1) RETURNING id", name).Scan(&chatId)
	if err != nil {
		return 0, err
	}

	userIDs, err := r.getUserIDsByUsernames(ctx, tx, memberUsernames)
	if err != nil {
		return 0, fmt.Errorf("failed to find users: %v", err)
	}

	for _, userID := range userIDs {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2)",
			chatId, userID)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return int(chatId), nil
}

func (r *ChatRepository) GetUserChats(ctx context.Context, username string) (*[]models.Chat, error) {
	var chats []models.Chat

	query := `
		SELECT 
			c.id, 
			c.chatname,
			cp.joined_at,
			ARRAY_AGG(u.username) as members
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id
		JOIN users u ON u.id = cp.user_id
		WHERE c.id IN (
			SELECT chat_id 
			FROM chat_participants cp2
			JOIN users u2 ON u2.id = cp2.user_id 
			WHERE u2.username = $1
		)
		GROUP BY c.id, c.chatname, cp.joined_at
		ORDER BY cp.joined_at DESC`

	rows, err := r.db.QueryContext(ctx, query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chat models.Chat
		var joinedAt sql.NullTime
		var members string // PostgreSQL reterns ARRAY_AGG like string

		err := rows.Scan(&chat.ID, &chat.Name, &joinedAt, &members)
		if err != nil {
			return nil, err
		}

		if joinedAt.Valid {
			chat.CreatedAt = joinedAt.Time
		}

		// the string witg array tooo []string
		// PostgreSQL reterns ARRAY_AGG in format: {user1,user2,user3}
		if members != "" {
			members = strings.Trim(members, "{}")
			if members != "" {
				chat.Members = strings.Split(members, ",")
			}
		}

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &chats, nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, chatID int) (*models.Chat, error) {
	var chat models.Chat
	var members string

	query := `
		SELECT 
			c.id, 
			c.chatname,
			ARRAY_AGG(u.username) as members
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id
		JOIN users u ON u.id = cp.user_id
		WHERE c.id = $1
		GROUP BY c.id, c.chatname`

	err := r.db.QueryRowContext(ctx, query, chatID).
		Scan(&chat.ID, &chat.Name, &members)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if members != "" {
		members = strings.Trim(members, "{}")
		if members != "" {
			chat.Members = strings.Split(members, ",")
		}
	}

	return &chat, nil
}

func (r *ChatRepository) DeleteChat(ctx context.Context, chatID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM chats WHERE id = $1", chatID)
	return err
}

func (r *ChatRepository) getChatMembers(ctx context.Context, chatID int) ([]string, error) {
	query := `
		SELECT u.username 
		FROM users u
		JOIN chat_participants cp ON u.id = cp.user_id
		WHERE cp.chat_id = $1`

	rows, err := r.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, err
		}
		members = append(members, username)
	}

	return members, nil
}

func (r *ChatRepository) getUserIDsByUsernames(ctx context.Context, tx *sql.Tx, usernames []string) ([]int, error) {
	if len(usernames) == 0 {
		return []int{}, nil
	}

	// Create placeholders for IN request
	placeholders := make([]string, len(usernames))
	args := make([]interface{}, len(usernames))
	for i, username := range usernames {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = username
	}

	query := fmt.Sprintf(`
		SELECT id 
		FROM users 
		WHERE username IN (%s)`, strings.Join(placeholders, ","))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) != len(usernames) {
		return nil, fmt.Errorf("some users not found")
	}

	return userIDs, nil
}

func (r *ChatRepository) getUsernameByID(ctx context.Context, userID int) (string, error) {
	var username string
	err := r.db.QueryRowContext(ctx, "SELECT username FROM users WHERE id = $1", userID).
		Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}
