package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	log *slog.Logger
	db  *redis.Client
}

type ConnectOptions struct {
	Addr     string
	Password string
	DB       int
}

const (
	chatKey     = "chat:"
	messagesKey = "messages:"
)

var (
	ErrNoConnection = errors.New("can't establish connection to db")
	ErrNoRows       = errors.New("can't find any rows")
	ErrInternal     = errors.New("internal error")
	ErrChatNotFound = errors.New("chat not found")
)

func New(log *slog.Logger, opt ConnectOptions) (*Redis, error) {
	db := redis.NewClient(&redis.Options{Addr: opt.Addr, Password: opt.Password, DB: opt.DB})

	_, err := db.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("can't ping Redis DB: %w", ErrNoConnection)
	}
	return &Redis{log: log, db: db}, nil
}

type Chat struct {
	Uuid     string
	Owner    string `redis:"user"`
	Readonly bool   `redis:"readonly"`
	Ttl      time.Duration
}

type Message struct {
	AuthorUuid uuid.UUID `json:"authorUuid"`
	Body       string    `json:"body"`
	Published  time.Time `json:"published"`
}

func (r *Redis) ChatsCount(ctx context.Context) (int, error) {
	op := "redis.ChatsCount"
	log := r.log.With(slog.String("op", op))

	rows, err := r.db.Keys(ctx, chatKey+"*").Result()
	if err != nil {
		log.Error("transaction error", sl.Err(err))
		return 0, ErrInternal
	}
	return len(rows), nil
}

func (r *Redis) CreateChat(ctx context.Context, chat domain.Chat) (*domain.Chat, error) {
	op := "redis.CreateChat"
	log := r.log.With(slog.String("op", op))

	redisChat := Chat{
		Uuid:     chat.Uuid.String(),
		Owner:    chat.Owner.Uuid.String(),
		Readonly: chat.Readonly,
		Ttl:      time.Duration(time.Until(chat.Deadline)),
	}

	_, err := r.db.HSet(ctx, chatKey+redisChat.Uuid, redisChat).Result()
	if err != nil {
		log.Error("HSET error in redis", sl.Err(err))
		return nil, ErrInternal
	}
	_, err = r.db.Expire(ctx, chatKey+redisChat.Uuid, redisChat.Ttl).Result()
	if err != nil {
		log.Error("EXPIRE error in redis", sl.Err(err))
		return nil, ErrInternal
	}

	return &chat, nil
}

func (r *Redis) GetChat(ctx context.Context, chatUuid uuid.UUID) (*domain.Chat, error) {
	op := "redis.GetChat"
	log := r.log.With(slog.String("op", op))

	var chat Chat
	err := r.db.HGetAll(ctx, chatKey+chatUuid.String()).Scan(&chat)
	if err != nil {
		log.Error("HGET readonly error", sl.Err(err))
		return nil, ErrInternal
	}

	uuid, err := uuid.Parse(chat.Owner)
	if err != nil {
		return nil, err
	}

	result := domain.Chat{
		Uuid:     chatUuid,
		Owner:    domain.User{Uuid: uuid},
		Readonly: chat.Readonly,
	}
	return &result, nil
}

func (r *Redis) PostMessage(ctx context.Context, chat uuid.UUID, message domain.Message) (*domain.Message, error) {
	op := "redis.PostMessage"
	log := r.log.With(slog.String("op", op))

	newMessage := Message{AuthorUuid: message.AuthorUuid, Body: message.Body, Published: message.Published}
	jsonMessage, err := json.Marshal(newMessage)
	if err != nil {
		log.Error("marshalling error", sl.Err(err))
		return nil, ErrInternal
	}

	r.db.LPush(ctx, messagesKey+chat.String(), jsonMessage)

	return nil, nil
}

func (r *Redis) TrimMessages(ctx context.Context, chat uuid.UUID, maximumMessages int) (bool, error) {
	op := "redis.TrimMessages"
	log := r.log.With(slog.String("op", op))

	_, err := r.db.LTrim(ctx, messagesKey+chat.String(), 0, int64(maximumMessages)-1).Result()
	if err != nil {
		log.Error("LTRIM error in redis", sl.Err(err))
		return false, ErrInternal
	}
	return true, nil
}

func (r *Redis) GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]*domain.Message, error) {
	op := "redis.GetChatHistory"
	log := r.log.With(slog.String("op", op))

	fmt.Println(messagesKey + chatUuid.String())
	messagesJson, err := r.db.LRange(ctx, messagesKey+chatUuid.String(), 0, -1).Result()
	if err != nil {
		log.Error("LRANGE error in redis", sl.Err(err))
		return nil, ErrInternal
	}

	var result []*domain.Message
	for _, v := range messagesJson {
		var message Message
		err := json.Unmarshal([]byte(v), &message)
		if err != nil {
			log.Error("unmarshall error", sl.Err(err))
			return nil, ErrInternal
		}
		result = append(result, &domain.Message{AuthorUuid: message.AuthorUuid, Body: message.Body, Published: message.Published})
	}
	return result, nil
}
