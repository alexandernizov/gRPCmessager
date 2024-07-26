package chat

import (
	"context"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
)

type ChatStorage interface {
	MakeNewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (uuid.UUID, error)
	PostMessage(ctx context.Context, authorUuid uuid.UUID, chat uuid.UUID, message string) (bool, error)
	GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]domain.Message, error)
}

type ChatService struct {
	log            *slog.Logger
	chatStorage    ChatStorage
	defaultChatTtl time.Duration
}

func NewChatService(log *slog.Logger, chatStorage ChatStorage, ttl time.Duration) *ChatService {
	return &ChatService{log: log, chatStorage: chatStorage, defaultChatTtl: ttl}
}

func (c *ChatService) NewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (uuid.UUID, error) {
	if ttl <= 0 {
		ttl = int(c.defaultChatTtl.Seconds())
	}
	uuid, err := c.chatStorage.MakeNewChat(ctx, ownerUuid, readonly, ttl)
	return uuid, err
}
func (c *ChatService) NewMessage(ctx context.Context, authorUuid uuid.UUID, chatUuid uuid.UUID, message string) (bool, error) {
	return c.chatStorage.PostMessage(ctx, authorUuid, chatUuid, message)
}

func (c *ChatService) ChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]domain.Message, error) {
	return c.chatStorage.GetChatHistory(ctx, chatUuid)
}
