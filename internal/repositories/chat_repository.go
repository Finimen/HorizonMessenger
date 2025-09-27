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

func (r *ChatRepository) CreateChat(ctx context.Context, name string, members []string) (int, error) {
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

	for _, member := range members {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO chat_participants (chat_id, user_id) VALUES (?, ?)",
			chatId, member)
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
	var chats []models.Chat

	query := `
		SELECT
			c.id,
			c.chatname, 
			cp.joined_at,
			(SELECT COUNT(*) FROM chat_participants cp2 WHERE cp2.chat_id = c.id) as participant_count
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = ?
		ORDER BY cp.joined_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chat models.Chat
		var joinedAt sql.NullTime
		var participantCount int

		err := rows.Scan(&chat.ID, &chat.Name, &joinedAt, &participantCount)
		if err != nil {
			return nil, err
		}

		if joinedAt.Valid {
			chat.CreatedAt = joinedAt.Time
		}
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

	rows, err := r.db.QueryContext(ctx,
		"SELECT user_id FROM chat_participants WHERE chat_id = ?", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		members = append(members, userID)
	}

	chat.Members = members
	return &chat, nil
}
