package chat

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/domain/errs"
	"github.com/google/uuid"
)

type ChatService struct {
	chatStorage    ChatStorage
	defaultChatTtl int
}

type ChatStorage interface {
	MakeNewChat(ctx context.Context, owner *domain.User, readonly bool, ttl int) (uuid.UUID, error)
	PostMessage(ctx context.Context, author *domain.User, chat uuid.UUID, message string) (bool, error)
	IsChatReadOnly(ctx context.Context, chatUuid uuid.UUID) (bool, *domain.User, error)
	GetChatHistory(ctx context.Context, chat uuid.UUID) ([]domain.Message, error)
	GetMessage(ctx context.Context, message uuid.UUID) (*domain.Message, error)
	UpdateMessage(ctx context.Context, message *domain.Message, newText string) (bool, error)
}

func NewChatService(chatStorage ChatStorage, defaultChatTTL int) *ChatService {
	return &ChatService{chatStorage: chatStorage, defaultChatTtl: defaultChatTTL}
}

func (c *ChatService) NewChat(ctx context.Context, owner *domain.User, readonly bool, ttl int) (uuid.UUID, error) {
	if ttl <= 0 {
		ttl = c.defaultChatTtl
	}
	res, err := c.chatStorage.MakeNewChat(ctx, owner, readonly, ttl)
	return res, err
}

func (c *ChatService) NewMessage(ctx context.Context, author *domain.User, chatUuid uuid.UUID, message string) (bool, error) {
	readOnly, owner, err := c.chatStorage.IsChatReadOnly(ctx, chatUuid)
	if err != nil {
		return false, err
	}

	if readOnly && (owner.Uuid != author.Uuid) {
		return false, errs.ErrOnlyOwnerCanPostMessage
	}

	res, err := c.chatStorage.PostMessage(ctx, author, chatUuid, message)
	if err != nil {
		return false, err
	}
	return res, nil
}

func (c *ChatService) ChatHistory(ctx context.Context, chat uuid.UUID) ([]domain.Message, error) {
	return c.chatStorage.GetChatHistory(ctx, chat)
}

func (c *ChatService) EditMessage(ctx context.Context, author *domain.User, messageUuid uuid.UUID, message string) (bool, error) {
	messageForEdit, err := c.chatStorage.GetMessage(ctx, messageUuid)
	if err != nil {
		return false, err
	}
	res, err := c.chatStorage.UpdateMessage(ctx, messageForEdit, message)
	if err != nil {
		return false, err
	}
	return res, nil
}
