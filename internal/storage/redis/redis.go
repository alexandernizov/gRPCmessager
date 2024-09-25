package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/api/gen/outbox"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
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
	chatKey        = "chat:"
	messagesKey    = "messages:"
	usersKey       = "users:"
	userLoginIndex = "userLoginIndex:"
	refreshTokens  = "refreshToken:"
	outboxList     = "outboxList:"
	outboxMessage  = "outboxMessage:"
)

func New(log *slog.Logger, opt ConnectOptions) (*Redis, error) {
	db := redis.NewClient(&redis.Options{Addr: opt.Addr, Password: opt.Password, DB: opt.DB})

	_, err := db.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("can't ping Redis DB: %w", storage.ErrNoConnection)
	}
	return &Redis{log: log, db: db}, nil
}

type User struct {
	Uuid         string `redis:"uuid"`
	Login        string `redis:"login"`
	PasswordHash []byte `redis:"password"`
}

type Chat struct {
	Uuid     string
	Owner    string `redis:"user"`
	Readonly bool   `redis:"readonly"`
	Ttl      time.Duration
}

type Message struct {
	Uuid       uuid.UUID `json:"uuid"`
	AuthorUuid uuid.UUID `json:"authorUuid"`
	Body       string    `json:"body"`
	Published  time.Time `json:"published"`
}

type OutboxMessage struct {
	Topic   string `redis:"topic"`
	Message []byte `redis:"message"`
}

func (r *Redis) ChatsCount(ctx context.Context) (int, error) {
	op := "redis.ChatsCount"
	log := r.log.With(slog.String("op", op))

	rows, err := r.db.Keys(ctx, chatKey+"*").Result()
	if err != nil {
		log.Error("transaction error", sl.Err(err))
		return 0, storage.ErrInternal
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

	outboxChat := outbox.OutboxChat{
		Uuid:      chat.Uuid.String(),
		OwnerUuid: chat.Owner.Uuid.String(),
		Readonly:  chat.Readonly,
		Deadline:  chat.Deadline.String(),
	}

	marshalledMessage, err := proto.Marshal(&outboxChat)
	if err != nil {
		return &domain.Chat{}, storage.ErrInternal
	}

	forSending := OutboxMessage{
		Topic:   domain.ChatTopic,
		Message: marshalledMessage,
	}

	pipe := r.db.TxPipeline()
	pipe.HSet(ctx, chatKey+redisChat.Uuid, redisChat)
	pipe.Expire(ctx, chatKey+redisChat.Uuid, redisChat.Ttl)
	pipe.RPush(ctx, outboxList, redisChat.Uuid)
	pipe.HSet(ctx, outboxMessage+redisChat.Uuid, forSending)
	_, err = pipe.Exec(ctx)

	if err != nil {
		log.Error("HSET error CREATE CHAT in redis", sl.Err(err))
		return nil, storage.ErrInternal
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
		return nil, storage.ErrInternal
	}

	ownerUuid, err := uuid.Parse(chat.Owner)
	if err != nil {
		return nil, storage.ErrChatNotFound
	}

	result := domain.Chat{
		Uuid:     chatUuid,
		Owner:    domain.User{Uuid: ownerUuid},
		Readonly: chat.Readonly,
	}
	return &result, nil
}

func (r *Redis) PostMessage(ctx context.Context, chat uuid.UUID, message domain.Message) (*domain.Message, error) {
	op := "redis.PostMessage"
	log := r.log.With(slog.String("op", op))

	mUuid := uuid.New()

	redisMessage := Message{Uuid: mUuid, AuthorUuid: message.AuthorUuid, Body: message.Body, Published: message.Published}
	jsonMessage, err := json.Marshal(redisMessage)
	if err != nil {
		log.Error("marshalling error", sl.Err(err))
		return nil, storage.ErrInternal
	}

	outboxMsg := outbox.OutboxMessage{
		Id:         1,
		AuthorUuid: message.AuthorUuid.String(),
		Body:       message.Body,
		Published:  message.Published.String(),
	}

	marshalledMessage, err := proto.Marshal(&outboxMsg)
	if err != nil {
		return &domain.Message{}, storage.ErrInternal
	}

	forSending := OutboxMessage{
		Topic:   domain.MessageTopic,
		Message: marshalledMessage,
	}

	pipe := r.db.TxPipeline()
	pipe.LPush(ctx, messagesKey+chat.String(), jsonMessage)
	pipe.RPush(ctx, outboxList, redisMessage.Uuid.String())
	pipe.HSet(ctx, outboxMessage+redisMessage.Uuid.String(), forSending)
	_, err = pipe.Exec(ctx)
	if err != nil {
		log.Error("HSET error POST MESSAGE in redis", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &message, nil
}

func (r *Redis) TrimMessages(ctx context.Context, chat uuid.UUID, maximumMessages int) (bool, error) {
	op := "redis.TrimMessages"
	log := r.log.With(slog.String("op", op))

	_, err := r.db.LTrim(ctx, messagesKey+chat.String(), 0, int64(maximumMessages)-1).Result()
	if err != nil {
		log.Error("LTRIM error in redis", sl.Err(err))
		return false, storage.ErrInternal
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
		return nil, storage.ErrInternal
	}

	var result []*domain.Message
	for _, v := range messagesJson {
		var message Message
		err := json.Unmarshal([]byte(v), &message)
		if err != nil {
			log.Error("unmarshall error", sl.Err(err))
			return nil, storage.ErrInternal
		}
		result = append(result, &domain.Message{AuthorUuid: message.AuthorUuid, Body: message.Body, Published: message.Published})
	}
	return result, nil
}

func (r *Redis) CreateUser(ctx context.Context, user domain.User) (*domain.User, error) {
	op := "redis.CreateUser"
	log := r.log.With(slog.String("op", op))

	redisUser := User{
		Uuid:         user.Uuid.String(),
		Login:        user.Login,
		PasswordHash: user.PasswordHash,
	}

	pipe := r.db.TxPipeline()
	pipe.HSet(ctx, usersKey+redisUser.Uuid, redisUser)
	pipe.Set(ctx, userLoginIndex+user.Login, user.Uuid.String(), -1)
	_, err := pipe.Exec(ctx)

	if err != nil {
		log.Error("HSET error in redis", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &user, nil
}

func (r *Redis) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	op := "redis.GetUserByLogin"
	log := r.log.With(slog.String("op", op))

	//1. Get UserUuid by Login
	userUuid, err := r.db.Get(ctx, userLoginIndex+login).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, storage.ErrUserNotFound
		}
		log.Error("GET UserUuid by Login error", sl.Err(err))
		return nil, storage.ErrInternal
	}
	parsedUuid, err := uuid.Parse(userUuid)
	if err != nil {
		log.Error("error in parsing uuid", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return r.GetUserByUuid(ctx, parsedUuid)
}

func (r *Redis) GetUserByUuid(ctx context.Context, userUuid uuid.UUID) (*domain.User, error) {
	op := "redis.GetUserByUuid"
	log := r.log.With(slog.String("op", op))

	var user User
	err := r.db.HGetAll(ctx, usersKey+userUuid.String()).Scan(&user)
	if err != nil {
		log.Error("GET User by UserUuid error", sl.Err(err))
		return nil, storage.ErrInternal
	}
	if user.Login == "" {
		return nil, storage.ErrUserNotFound
	}

	parsedUuid, err := uuid.Parse(user.Uuid)
	if err != nil {
		log.Error("failed to parse uuid", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &domain.User{Uuid: parsedUuid, Login: user.Login, PasswordHash: user.PasswordHash}, nil
}

func (r *Redis) UpsertRefreshToken(ctx context.Context, userUuid uuid.UUID, refreshToken string) error {
	op := "redis.UpsertRefresToken"
	log := r.log.With(slog.String("op", op))

	pipe := r.db.TxPipeline()
	pipe.Set(ctx, refreshTokens+userUuid.String(), refreshToken, -1)
	_, err := pipe.Exec(ctx)

	if err != nil {
		log.Error("HSET error in redis", sl.Err(err))
		return storage.ErrInternal
	}
	return nil
}

func (r *Redis) GetRefreshToken(ctx context.Context, userUuid uuid.UUID) (string, error) {
	op := "redis.GetRefreshToken"
	log := r.log.With(slog.String("op", op))

	//1. Get UserUuid by Login
	token, err := r.db.Get(ctx, refreshTokens+userUuid.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", storage.ErrTokenNotFound
		}
		log.Error("GET UserUuid by Login error", sl.Err(err))
		return "", storage.ErrInternal
	}
	return token, nil
}

func (r *Redis) GetNextOutbox(ctx context.Context) (*domain.Outbox, error) {
	op := "redis.GetNextOutbox"
	log := r.log.With(slog.String("op", op))

	outboxUuid, err := r.db.LIndex(ctx, outboxList, 0).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, storage.ErrNoOutbox
		}
		log.Error("error in getting next outbox", sl.Err(err))
		return nil, storage.ErrInternal
	}

	forSending := OutboxMessage{}

	err = r.db.HGetAll(ctx, outboxMessage+outboxUuid).Scan(&forSending)

	if err != nil {
		log.Error("error unmarshalling message", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &domain.Outbox{Uuid: uuid.MustParse(outboxUuid), Topic: forSending.Topic, Message: forSending.Message}, nil
}

func (r *Redis) ConfirmOutboxSended(ctx context.Context, outboxUuid uuid.UUID) error {
	// op := "redis.ConfirmOutboxSended"
	// log := r.log.With(slog.String("op", op))

	r.db.LRem(ctx, outboxList, 1, outboxUuid.String())
	r.db.HDel(ctx, outboxMessage+outboxUuid.String())

	return nil
}
