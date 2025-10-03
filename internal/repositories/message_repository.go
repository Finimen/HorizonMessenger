package repositories

import (
	"context"
	"database/sql"
	_ "embed"
	"log/slog"
	"massager/internal/models"
)

//go:embed migrations/005_create_messages_table_up.sql
var createMessageTableQuery string

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB, logger *slog.Logger) (*MessageRepository, error) {
	var repo = MessageRepository{db: db}
	var _, err = repo.db.Exec(string(createMessageTableQuery))
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	logger.Info("messege initialization: stage 1")

	err = db.Ping()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	logger.Info("messege initialization: stage 2")

	return &repo, nil
}

func (r *MessageRepository) CreateMessage(ctx context.Context, senderName, content string, chatID int) error {
	var userId int
	var rowId = r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE username = $1", senderName)
	err := rowId.Scan(&userId)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, "INSERT INTO messages (chat_id, sender_id, message_content) VALUES ($1, $2, $3)", chatID, userId, content)
	if err != nil {
		return err
	}
	return nil
}

func (r *MessageRepository) GetMessages(ctx context.Context, chatID, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT 
			u.username,
			m.message_content,
			m.created_at,
			c.chatname,
			m.chat_id
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		JOIN chats c ON m.chat_id = c.id
		WHERE m.chat_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var message models.Message

		err = rows.Scan(&message.Sender, &message.Content, &message.Timestamp, &message.ChatName, &message.ChatID)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *MessageRepository) DeleteMessagesByChatID(ctx context.Context, chatID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages WHERE chat_id = $1", chatID)
	return err
}

func (r *MessageRepository) UpdateMessage(ctx context.Context, messageID int, newContent string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE messages SET message_content = $1 WHERE id = $2",
		newContent, messageID)
	return err
}

func (r *MessageRepository) DeleteMessage(ctx context.Context, messageID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages WHERE id = $1", messageID)
	return err
}
