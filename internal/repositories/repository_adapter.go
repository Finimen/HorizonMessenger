package repositories

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type RepositoryAdapter struct {
	User    *UserRepository
	Chat    *ChatRepository
	Message *MessageRepository
}

func NewRepositoryAdapter(path string) (*RepositoryAdapter, error) {
	var db, e = sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}

	fmt.Println("STAGE 1")

	if err := db.Ping(); err != nil {
		return nil, err
	}

	userRepo, err1 := NewUserRepository(db)
	if err1 != nil {
		return nil, e
	}

	fmt.Println("STAGE 2")

	var chatRepo, err2 = NewChatRepository(db)
	if err2 != nil {
		return nil, e
	}
	var messageRepo, err3 = NewMessageRepository(db)
	if err3 != nil {
		return nil, e
	}
	fmt.Println("STAGE 3")

	return &RepositoryAdapter{User: userRepo, Message: messageRepo, Chat: chatRepo}, nil
}
