package ports

import (
	"context"
	"massager/internal/models"
)

type IUserRepository interface {
	IUserRepositoryReader
	IUserRepositoryWriter
}

type IUserRepositoryReader interface {
	GetUserByName(context.Context, string) (*models.User, error)
}

type IUserRepositoryWriter interface {
	CreateUser(context.Context, string, string, string) error
}
