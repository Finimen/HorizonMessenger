package ports

import (
	"context"
	"time"
)

type TokenRepository interface {
	IsRevoked(ctx context.Context, tokenHash string) (bool, error)
	Revoke(ctx context.Context, tokenHash string, expiration time.Duration) error
}
