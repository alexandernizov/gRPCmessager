package redis

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	log               *slog.Logger
	db                *redis.Client
	MaxChatsCount     int
	MaxMessagesInChat int
}

func NewRedisChatStorage(
	log *slog.Logger,
	addr string,
	port string,
	password string,
	db int,
	maxChatsCount int,
	maxMessagesInChat int,
) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr + ":" + port,
		Password: password,
		DB:       db,
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &Redis{log: log, db: client, MaxChatsCount: maxChatsCount, MaxMessagesInChat: maxMessagesInChat}, nil
}

func (r *Redis) MakeNewChat(ctx context.Context, ownerUuid uuid.UUID, readonly bool, ttl int) (uuid.UUID, error) {
	//Check how many chats we already have in Reddis

	chats, _, err := r.db.Scan(ctx, 0, "chat:*", int64(r.MaxChatsCount)).Result()
	if err != nil {
		return uuid.Nil, err
	}
	if len(chats) >= r.MaxChatsCount {
		return uuid.Nil, errors.New("maximum chats reached")
	}

	newChatUuid := uuid.New()
	deadline := time.Now().Add((time.Duration(ttl) * time.Second)).Unix()
	//Make new Chat
	newChat := domain.Chat{Uuid: newChatUuid.String(), OwnerUuid: ownerUuid.String(), Readonly: readonly, Deadline: deadline}
	r.db.HSet(ctx, "chat:"+newChatUuid.String(), newChat)

	return newChatUuid, nil
}

func (r *Redis) PostMessage(ctx context.Context, authorUuid uuid.UUID, chat uuid.UUID, body string) (bool, error) {
	//Try to get Chat
	var c domain.Chat
	res := r.db.HGetAll(ctx, "chat:"+chat.String())
	res.Scan(&c)

	if c.Uuid == "" {
		return false, errors.New("has no such chat")
	}

	if c.Readonly {
		if c.OwnerUuid != authorUuid.String() {
			return false, errors.New("has no permission to post message in this chat")
		}
	}

	//GetMessagesCount
	messages, _, err := r.db.Scan(ctx, 0, "message:"+c.Uuid+":*", int64(r.MaxChatsCount)).Result()
	if err != nil {
		return false, err
	}
	if len(messages) >= r.MaxMessagesInChat {
		return false, errors.New("maximum messages in this chat reached")
	}

	message := domain.Message{Uuid: uuid.New().String(), AuthorUuid: authorUuid.String(), Body: body, Published: time.Now().Unix()}
	r.db.HSet(ctx, "message:"+c.Uuid+":"+message.Uuid, message)
	return true, nil
}

func (r *Redis) GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]domain.Message, error) {
	var res []domain.Message
	messagesKeys, _, err := r.db.Scan(ctx, 0, "message:"+chatUuid.String()+":*", int64(r.MaxChatsCount)).Result()
	if err != nil {
		return res, err
	}

	for _, key := range messagesKeys {
		var message domain.Message
		r.db.HGetAll(ctx, key).Scan(&message)
		res = append(res, message)
	}

	return res, nil
}
