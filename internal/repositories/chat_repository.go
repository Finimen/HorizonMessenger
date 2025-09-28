package repositories

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"massager/internal/models"
)

//go:embed migrations/003_create_chats_table_up.sql
var createChatTableQuery string

//go:embed migrations/004_create_chat_participants_up.sql
var createСhatParticipantsQuery string

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) (*ChatRepository, error) {
	var repo = ChatRepository{db: db}
	var _, err = repo.db.Exec(string(createChatTableQuery))
	_, err = repo.db.Exec(string(createСhatParticipantsQuery))
	if err != nil {
		return nil, err
	}

	fmt.Println("CHAT STAGE 1")

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("CHAT STAGE 2")

	return &repo, nil
}

func (r *ChatRepository) CreateChat(ctx context.Context, name string, memberUsernames []string) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "INSERT INTO chats (chatname) VALUES (?)", name)
	if err != nil {
		return 0, err
	}

	chatId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, username := range memberUsernames {
		userID, err := r.getUserIDByUsername(ctx, username)
		if err != nil {
			return 0, fmt.Errorf("failed to find user %s: %v", username, err)
		}

		_, err = tx.ExecContext(ctx,
			"INSERT INTO chat_participants (chat_id, user_id) VALUES (?, ?)",
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

func (r *ChatRepository) GetUserChats(ctx context.Context, userID string) (*[]models.Chat, error) {
	numericUserID, err := r.getUserIDByUsername(ctx, userID)
	if err != nil {
		return nil, err
	}

	var chats []models.Chat

	query := `
        SELECT
            c.id,
            c.chatname, 
            cp.joined_at
        FROM chats c
        JOIN chat_participants cp ON c.id = cp.chat_id
        WHERE cp.user_id = ?
        ORDER BY cp.joined_at DESC`

	rows, err := r.db.QueryContext(ctx, query, numericUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chat models.Chat
		var joinedAt sql.NullTime

		err := rows.Scan(&chat.ID, &chat.Name, &joinedAt)
		if err != nil {
			return nil, err
		}

		if joinedAt.Valid {
			chat.CreatedAt = joinedAt.Time
		}

		members, err := r.getChatMembers(ctx, chat.ID)
		if err != nil {
			return nil, err
		}
		chat.Members = members

		chats = append(chats, chat)
	}

	return &chats, nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, chatID int) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.QueryRowContext(ctx, "SELECT id, chatname FROM chats WHERE id = ?", chatID).
		Scan(&chat.ID, &chat.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	members, err := r.getChatMembers(ctx, chatID)
	if err != nil {
		return nil, err
	}
	chat.Members = members

	return &chat, nil
}

func (r *ChatRepository) DeleteChat(ctx context.Context, chatID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM chats WHERE id = ?", chatID)
	return err
}

func (r *ChatRepository) getChatMembers(ctx context.Context, chatID int) ([]string, error) {
	query := `
        SELECT u.username 
        FROM users u
        JOIN chat_participants cp ON u.id = cp.user_id
        WHERE cp.chat_id = ?`

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

func (r *ChatRepository) getUserIDByUsername(ctx context.Context, username string) (int, error) {
	var userID int
	err := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE username = ?", username).
		Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("user not found: %s", username)
		}
		return 0, err
	}
	return userID, nil
}

func (r *ChatRepository) getUsernameByID(ctx context.Context, userID int) (string, error) {
	var username string
	err := r.db.QueryRowContext(ctx, "SELECT username FROM users WHERE id = ?", userID).
		Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}
