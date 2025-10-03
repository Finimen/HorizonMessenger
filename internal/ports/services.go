package ports

import (
	"context"
)

type IMessageService interface {
	SendMessage(ctx context.Context, senderID, content string, chatID int) error
}

type IEmailService interface {
	SendVerificationEmail(email, token string) error
}

type IHasher interface {
	GenerateFromPassword(password []byte, cost int) ([]byte, error)
	CompareHashAndPassword(storedPaswsord []byte, userPassword []byte) error
	DefaultCost() int
}
