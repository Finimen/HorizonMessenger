package repositories

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"massager/internal/models"
)

//go:embed migrations/005_create_messages_table_up.sql
var createMessageTableQuery string

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) (*MessageRepository, error) {
	var repo = MessageRepository{db: db}
	var _, err = repo.db.Exec(string(createMessageTableQuery))
	if err != nil {
		return nil, err
	}

	fmt.Println("MESSAGE STAGE 1")

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	fmt.Println("MESSAGE STAGE 2")

	return &repo, nil
}

func (r *MessageRepository) CreateMessage(ctx context.Context, senderName, content string, chatID int) error {
	var userId int
	var rowId = r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE username = ?", senderName)
	err := rowId.Scan(&userId)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, "INSERT INTO messages (chat_id, sender_id, message_content) VALUES (?, ?, ?)", chatID, userId, content)
	if err != nil {
		return err
	}
	return nil
}

func (r *MessageRepository) GetMessages(ctx context.Context, chatID, limit, offset int) ([]models.Message, error) {
	var rows, err = r.db.QueryContext(ctx, "SELECT sender_id, message_content,created_at FROM messages WHERE chat_id = ?", chatID)
	if err != nil {
		return nil, err
	}

	var messages []models.Message
	for rows.Next() {
		var message models.Message
		var id int

		message.ChatID = chatID
		err = rows.Scan(&id, &message.Content, &message.Timestamp)
		if err != nil {
			return nil, err
		}
		var rowId = r.db.QueryRowContext(ctx, "SELECT username FROM users WHERE id = ?", id)
		err = rowId.Scan(&message.Sender)
		if err != nil {
			return nil, err
		}
		var rowChat = r.db.QueryRowContext(ctx, "SELECT chatname FROM chats WHERE id = ?", chatID)
		err = rowChat.Scan(&message.ChatName)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	fmt.Println("LENLEN___ ", len(messages), "___LENLEN")
	return messages, nil
}

func (r *MessageRepository) DeleteMessagesByChatID(ctx context.Context, chatID int) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM messages WHERE chat_id = ?", chatID)
	return err
}
