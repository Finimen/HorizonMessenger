package repositories

import "context"

type MessageRepository struct {
}

type Message struct {
}

func (r *MessageRepository) Create(context.Context, string, string, string) error {
	return nil
}

func (r *MessageRepository) GetConversation(context.Context, string, string, int, int) ([]Message, error) {
	return nil, nil
}
