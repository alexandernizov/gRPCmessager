package chatstorage

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
)

type CStorage struct {
	ChatStorage
}

func NewStorage(chatStorage ChatStorage) *CStorage {
	return &CStorage{chatStorage}
}

type ChatStorage interface {
	MakeNewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (uuid.UUID, error)
	PostMessage(ctx context.Context, authorUuid uuid.UUID, chat uuid.UUID, message string) (bool, error)
	GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]domain.Message, error)
}
