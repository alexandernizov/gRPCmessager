package authstorage

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
)

type AStorage struct {
	AuthStorage
}

func NewStorage(authStorage AuthStorage) *AStorage {
	return &AStorage{authStorage}
}

type AuthStorage interface {
	NewUser(ctx context.Context, user domain.User) (bool, error)
	GetUser(ctx context.Context, login string) (domain.User, error)
}
