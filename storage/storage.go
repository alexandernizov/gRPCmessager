package storage

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
)

type Storage struct {
	AuthStorage
	ChatStorage
}

func NewStorage(authStorage AuthStorage, chatStorage ChatStorage) *Storage {
	return &Storage{authStorage, chatStorage}
}

type AuthStorage interface {
	SaveUser(ctx context.Context, name string, password string) (bool, error)
	GetUser(ctx context.Context, name string, password string) (*domain.User, error)
	MakeUserSession(ctx context.Context, user *domain.User) (uuid.UUID, error)
	GetSession(ctx context.Context, user *domain.User) (*domain.Session, error)
}

type ChatStorage interface {
	MakeNewChat(ctx context.Context, owner *domain.User, readonly bool, ttl int) (uuid.UUID, error)
	PostMessage(ctx context.Context, author *domain.User, chat uuid.UUID, message string) (bool, error)
	IsChatReadOnly(ctx context.Context, chatUuid uuid.UUID) (bool, *domain.User, error)
	GetChatHistory(ctx context.Context, chat uuid.UUID) ([]domain.Message, error)
	GetMessage(ctx context.Context, message uuid.UUID) (*domain.Message, error)
	UpdateMessage(ctx context.Context, message *domain.Message, newText string) (bool, error)
}
