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
