package chat

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage/redis"
	"github.com/google/uuid"
)

type ChatStorage interface {
	CreateChat(ctx context.Context, chat domain.Chat) (*domain.Chat, error)
	GetChat(ctx context.Context, chatUuid uuid.UUID) (*domain.Chat, error)
	ChatsCount(ctx context.Context) (int, error)
	PostMessage(ctx context.Context, chat uuid.UUID, message domain.Message) (*domain.Message, error)
	TrimMessages(ctx context.Context, chat uuid.UUID, maximumMessages int) (bool, error)
	GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]*domain.Message, error)
}

type ChatNotifier interface {
	CreateChatOutbox(ctx context.Context, chat domain.Chat) error
}

var (
	ErrInternal               = errors.New("internal error")
	ErrMaximumChats           = errors.New("maximum chats created already")
	ErrPermissionDenied       = errors.New("have no permission for this operation")
	ErrChatNotFound           = errors.New("chat not found")
	ErrNotificationNotCreated = errors.New("notification was not created")
)

type ChatService struct {
	log          *slog.Logger
	chatOptions  ChatOptions
	chatStorage  ChatStorage
	chatNotifier ChatNotifier
}

type ChatOptions struct {
	DefaultTtl      time.Duration
	MaximumCount    int
	MaximumMessages int
}

func New(log *slog.Logger, chatOptions ChatOptions, chatStorage ChatStorage, chatNotifier ChatNotifier) *ChatService {
	return &ChatService{log: log, chatOptions: chatOptions, chatStorage: chatStorage, chatNotifier: chatNotifier}
}

func (c *ChatService) NewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (*domain.Chat, error) {
	// Check how many chats we have already
	chatsCount, err := c.chatStorage.ChatsCount(ctx)
	if err != nil {
		return nil, ErrInternal
	}
	if chatsCount >= c.chatOptions.MaximumCount {
		return nil, ErrMaximumChats
	}
	// Create new chat
	if ttl == 0 {
		ttl = int(c.chatOptions.DefaultTtl.Seconds())
	}
	newChat := domain.Chat{
		Uuid:     uuid.New(),
		Owner:    domain.User{Uuid: ownerUuid},
		Readonly: readonly,
		Deadline: time.Now().Add(time.Duration(time.Duration(ttl) * time.Second)),
	}

	createdChat, err := c.chatStorage.CreateChat(ctx, newChat)
	if err != nil {
		return nil, ErrInternal
	}

	err = c.chatNotifier.CreateChatOutbox(ctx, newChat)
	if err != nil {
		return createdChat, ErrNotificationNotCreated
	}

	return createdChat, nil
}

func (c *ChatService) NewMessage(ctx context.Context, chatUuid uuid.UUID, authorUuid uuid.UUID, message string) (*domain.Message, error) {
	newMessage := domain.Message{AuthorUuid: authorUuid, Body: message, Published: time.Now()}
	chat, err := c.chatStorage.GetChat(ctx, chatUuid)
	if err != nil {
		if errors.Is(err, redis.ErrChatNotFound) {
			return nil, ErrChatNotFound
		}
		return nil, ErrInternal
	}
	if chat.Readonly && chat.Owner.Uuid != authorUuid {
		return nil, ErrPermissionDenied
	}
	createdMessage, err := c.chatStorage.PostMessage(ctx, chatUuid, newMessage)
	if err != nil {
		return nil, ErrInternal
	}
	_, err = c.chatStorage.TrimMessages(ctx, chatUuid, c.chatOptions.MaximumMessages)
	if err != nil {
		c.log.Error("error with during messages triming", sl.Err(err))
	}
	return createdMessage, nil
}

func (c *ChatService) ChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]*domain.Message, error) {
	res, err := c.chatStorage.GetChatHistory(ctx, chatUuid)
	if err != nil {
		return nil, ErrInternal
	}
	return res, nil
}
