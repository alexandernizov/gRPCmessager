package inmemory

import (
	"context"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/storage"
	"github.com/google/uuid"

	"github.com/alexandernizov/grpcmessanger/api/gen/outbox"
	"google.golang.org/protobuf/proto"
)

type Inmemory struct {
	log *slog.Logger

	users         []User
	refreshTokens []RefreshToken
	chats         []Chat
	messages      []Message

	outboxes []Outbox
}

func New(log *slog.Logger) *Inmemory {
	return &Inmemory{log: log}
}

type Outbox struct {
	uuid    uuid.UUID
	topic   string
	message []byte
	sent_at time.Time
}

type User struct {
	Uuid         uuid.UUID
	Login        string
	PasswordHash []byte
}

type RefreshToken struct {
	userUuid     uuid.UUID
	refreshToken string
}

type Chat struct {
	Uuid     uuid.UUID
	Owner    uuid.UUID
	Readonly bool
	Deadline time.Time
}

type Message struct {
	Id         int
	ChatUuid   uuid.UUID
	AuthorUuid uuid.UUID
	Body       string
	Published  time.Time
}

type numerator struct {
	current int
}

func (n *numerator) GetNext() int {
	n.current = n.current + 1
	return n.current
}

var singleNumerator *numerator

func getNumerator() *numerator {
	if singleNumerator == nil {
		singleNumerator = &numerator{}
	}
	return singleNumerator
}

func (i *Inmemory) CreateUser(ctx context.Context, user domain.User) (*domain.User, error) {
	newUser := User{
		Uuid:         user.Uuid,
		Login:        user.Login,
		PasswordHash: user.PasswordHash,
	}
	i.users = append(i.users, newUser)
	return &user, nil
}

func (i *Inmemory) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	var user domain.User
	for _, v := range i.users {
		if v.Login == login {
			user.Uuid = v.Uuid
			user.PasswordHash = v.PasswordHash

			return &user, nil
		}
	}
	return &user, storage.ErrUserNotFound
}

func (i *Inmemory) GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*domain.User, error) {
	var user domain.User
	for _, v := range i.users {
		if v.Uuid == uuid {
			user.Login = v.Login
			user.PasswordHash = v.PasswordHash

			return &user, nil
		}
	}
	return &user, storage.ErrUserNotFound
}

func (i *Inmemory) UpsertRefreshToken(ctx context.Context, userUuid uuid.UUID, refreshToken string) error {
	for key := range i.refreshTokens {
		if i.refreshTokens[key].userUuid == userUuid {
			i.refreshTokens[key].refreshToken = refreshToken

			return nil
		}
	}
	newRefreshToken := RefreshToken{userUuid: userUuid, refreshToken: refreshToken}
	i.refreshTokens = append(i.refreshTokens, newRefreshToken)
	return nil
}

func (i *Inmemory) GetRefreshToken(ctx context.Context, userUuid uuid.UUID) (string, error) {
	for _, v := range i.refreshTokens {
		if v.userUuid == userUuid {
			return v.refreshToken, nil
		}
	}
	return "", storage.ErrTokenNotFound
}

func (i *Inmemory) CreateChat(ctx context.Context, chat domain.Chat) (*domain.Chat, error) {
	newChat := Chat{Uuid: chat.Uuid, Owner: chat.Owner.Uuid, Readonly: chat.Readonly, Deadline: chat.Deadline}

	msg := outbox.OutboxChat{
		Uuid:      newChat.Uuid.String(),
		OwnerUuid: newChat.Owner.String(),
		Readonly:  newChat.Readonly,
		Deadline:  newChat.Deadline.String(),
	}

	marshalledMessage, err := proto.Marshal(&msg)
	if err != nil {
		return &domain.Chat{}, storage.ErrInternal
	}

	i.chats = append(i.chats, newChat)
	i.outboxes = append(i.outboxes, Outbox{uuid: uuid.New(), topic: "Chats", message: marshalledMessage})

	return &chat, nil
}

func (i *Inmemory) GetChat(ctx context.Context, chatUuid uuid.UUID) (*domain.Chat, error) {
	var chat Chat
	for _, v := range i.chats {
		if v.Uuid == chatUuid {
			chat.Uuid = v.Uuid
			chat.Readonly = v.Readonly
			chat.Owner = v.Owner
			chat.Deadline = v.Deadline

			break
		}
	}
	var owner User
	if chat.Uuid != uuid.Nil {
		for _, v := range i.users {
			if v.Uuid == chat.Owner {
				owner.Uuid = v.Uuid
				owner.Login = v.Login
				owner.PasswordHash = v.PasswordHash

				break
			}
		}
	}
	res := domain.Chat{}
	if chat.Uuid != uuid.Nil && owner.Uuid != uuid.Nil {
		res.Uuid = chat.Uuid
		res.Readonly = chat.Readonly
		res.Deadline = chat.Deadline
		res.Owner.Uuid = owner.Uuid
		res.Owner.Login = owner.Login
		res.Owner.PasswordHash = owner.PasswordHash
		return &res, nil
	}
	return &res, storage.ErrChatNotFound
}

func (i *Inmemory) ChatsCount(ctx context.Context) (int, error) {
	return len(i.chats), nil
}

func (i *Inmemory) PostMessage(ctx context.Context, chat uuid.UUID, message domain.Message) (*domain.Message, error) {
	nextId := getNumerator().GetNext()
	newMessage := Message{Id: nextId, AuthorUuid: message.AuthorUuid, Body: message.Body, Published: message.Published, ChatUuid: chat}

	msg := outbox.OutboxMessage{
		Id:         int64(newMessage.Id),
		AuthorUuid: newMessage.AuthorUuid.String(),
		Body:       newMessage.Body,
		Published:  newMessage.Published.String(),
	}

	marshalledMessage, err := proto.Marshal(&msg)
	if err != nil {
		return &domain.Message{}, storage.ErrInternal
	}

	i.messages = append(i.messages, newMessage)
	i.outboxes = append(i.outboxes, Outbox{uuid: uuid.New(), topic: "Messages", message: marshalledMessage})

	return &domain.Message{Id: newMessage.Id, AuthorUuid: newMessage.AuthorUuid, Body: newMessage.Body, Published: newMessage.Published}, nil
}

func (i *Inmemory) TrimMessages(ctx context.Context, chat uuid.UUID, maximumMessages int) (bool, error) {
	if len(i.messages) > maximumMessages {
		i.messages = i.messages[len(i.messages)-maximumMessages:]
	}
	return true, nil
}

func (i *Inmemory) GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]*domain.Message, error) {
	var res []*domain.Message
	for _, v := range i.messages {
		if v.ChatUuid == chatUuid {
			nextMessage := domain.Message{Id: v.Id, AuthorUuid: v.AuthorUuid, Body: v.Body, Published: v.Published}
			res = append(res, &nextMessage)
		}
	}
	return res, nil
}

func (i *Inmemory) GetNextOutbox(ctx context.Context) (*domain.Outbox, error) {
	for _, v := range i.outboxes {
		if v.sent_at.IsZero() {
			return &domain.Outbox{Uuid: v.uuid, Topic: v.topic, Message: v.message, Sent_at: v.sent_at}, nil
		}
	}
	return nil, storage.ErrNoOutbox
}

func (i *Inmemory) ConfirmOutboxSended(ctx context.Context, outboxUuid uuid.UUID) error {
	for key := range i.outboxes {
		if i.outboxes[key].uuid == outboxUuid {
			i.outboxes[key].sent_at = time.Now()
		}
	}
	return nil
}
