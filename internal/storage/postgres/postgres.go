package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Postgres struct {
	log *slog.Logger
	db  *sql.DB
}

type ConnectOptions struct {
	Host     string
	Port     string
	User     string
	Password string
	DBname   string
}

var (
	ErrNoConnection     = errors.New("can't establish connection to db")
	ErrInternal         = errors.New("internal error")
	ErrUserNotFound     = errors.New("user is not found")
	ErrTokenNotFound    = errors.New("token is not found")
	ErrNoOutboxChats    = errors.New("have no outbox chats to send")
	ErrNoOutboxMessages = errors.New("have no outbox messages to send")
)

const (
	usersTable          = "users"
	refreshTokensTable  = "refresh_tokens"
	outboxChatsTable    = "outbox_chats"
	outboxMessagesTable = "outbox_messages"
)

func New(log *slog.Logger, db *sql.DB) *Postgres {
	return &Postgres{log, db}
}

func NewWithOptions(log *slog.Logger, opt ConnectOptions) (*Postgres, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		opt.Host,
		opt.Port,
		opt.User,
		opt.Password,
		opt.DBname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("can't open Postgres DB: %w", ErrNoConnection)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("can't ping Postgres DB: %w", ErrNoConnection)
	}

	return &Postgres{log: log, db: db}, nil
}

func (p *Postgres) Close() error {
	err := p.db.Close()
	if err != nil {
		return err
	}
	return nil
}

type User struct {
	Uuid         uuid.UUID `pg:"uuid"`
	Login        string    `pg:"login"`
	PasswordHash []byte    `pg:"password"`
}

type Chat struct {
	Id       int        `pg:"uuid"`
	Chat     uuid.UUID  `pg:"chat"`
	Author   uuid.UUID  `pg:"author"`
	ReadOnly bool       `pg:"read_only"`
	SentAt   *time.Time `pg:"sent_to_kafka"`
}

type txKey struct{}

func injectTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func (p *Postgres) extractTx(ctx context.Context) (tx *sql.Tx, closeTx func(err error)) {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx, func(err error) {}
	}

	tx, _ = p.db.Begin()
	return tx, func(err error) {
		if err != nil {
			errRollback := tx.Rollback()
			if errRollback != nil {
				p.log.Error("error according rollback transaction in DB", sl.Err(errRollback))
			}
			return
		}
		errCommit := tx.Commit()
		if errCommit != nil {
			p.log.Error("error according commit transaction in DB", sl.Err(errCommit))
		}
	}
}

func (p *Postgres) WithTx(ctx context.Context, tFunc func(ctx context.Context) error) error {
	op := "postgres.WithTx"
	log := p.log.With(slog.String("op", op))

	tx, beginError := p.db.Begin()
	if beginError != nil {
		log.Error("error with Start transaction", sl.Err(beginError))
		return ErrInternal
	}

	ctxTx := injectTx(ctx, tx)

	fnError := tFunc(ctxTx)

	if fnError != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Error("error with Rollback transaction", sl.Err(rollbackErr))
			return ErrInternal
		}
		return fnError
	}

	if commitError := tx.Commit(); commitError != nil {
		log.Error("error with Commit transaction", sl.Err(commitError))
		return ErrInternal
	}

	return nil
}

func (p *Postgres) CreateUser(ctx context.Context, user domain.User) (*domain.User, error) {
	const op = "postgres.CreateUser"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	pgUser := User{Uuid: user.Uuid, Login: user.Login, PasswordHash: user.PasswordHash}

	query := fmt.Sprintf("INSERT INTO %s (uuid, login, password) VALUES ($1,$2,$3)", usersTable)
	_, err := tx.Exec(query, pgUser.Uuid, pgUser.Login, pgUser.PasswordHash)
	closeTx(err)

	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return nil, ErrInternal
	}

	return &user, nil
}

func (p *Postgres) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	const op = "postgres.GetUserByLogin"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	pgUser := User{Login: login}

	query := fmt.Sprintf("SELECT uuid, login, password FROM %s WHERE users.login = $1", usersTable)
	row := tx.QueryRow(query, pgUser.Login)
	err := row.Scan(&pgUser.Uuid, &pgUser.Login, &pgUser.PasswordHash)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, ErrInternal
	}

	return &domain.User{Uuid: pgUser.Uuid, Login: pgUser.Login, PasswordHash: pgUser.PasswordHash}, nil
}

func (p *Postgres) GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*domain.User, error) {
	const op = "postgres.GetUserByLogin"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	pgUser := User{Uuid: uuid}

	query := fmt.Sprintf("SELECT uuid, login, password FROM %s WHERE users.uuid = $1", usersTable)
	row := tx.QueryRow(query, pgUser.Uuid)
	err := row.Scan(&pgUser.Uuid, &pgUser.Login, &pgUser.PasswordHash)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, ErrInternal
	}

	return &domain.User{Uuid: pgUser.Uuid, Login: pgUser.Login, PasswordHash: pgUser.PasswordHash}, nil
}

func (p *Postgres) UpsertRefreshToken(ctx context.Context, userUuid uuid.UUID, refreshToken string) error {
	const op = "postgres.UpsertRefreshToken"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := fmt.Sprintf("INSERT INTO %s (user_uuid, token) VALUES($1, $2) ON CONFLICT (user_uuid) DO UPDATE SET token = ($2)", refreshTokensTable)
	_, err := tx.Exec(query, userUuid, refreshToken)
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return ErrInternal
	}

	return nil
}

func (p *Postgres) GetRefreshToken(ctx context.Context, userUuid uuid.UUID) (string, error) {
	const op = "postgres.GetRefreshToken"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	var token string

	query := fmt.Sprintf("SELECT token FROM %s WHERE user_uuid = $1", refreshTokensTable)
	row := tx.QueryRow(query, userUuid)
	err := row.Scan(&token)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrTokenNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return "", ErrInternal
	}

	return token, nil
}

func (p *Postgres) CreateChatOutbox(ctx context.Context, chat domain.Chat) error {
	const op = "postgres.CreateChatOutbox"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := fmt.Sprintf("INSERT INTO %s (chat, author, read_only) VALUES ($1,$2,$3)", outboxChatsTable)
	_, err := tx.Exec(query, chat.Uuid, chat.Owner.Uuid, chat.Readonly)
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return ErrInternal
	}

	return nil
}

func (p *Postgres) GetNextOutboxChat(ctx context.Context) (*domain.Chat, error) {
	const op = "postgres.GetNextOutboxChat"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	pgChat := Chat{}

	query := fmt.Sprintf("SELECT chat, author, read_only, sent_to_kafka FROM %s WHERE sent_to_kafka IS NULL LIMIT 1", outboxChatsTable)
	row := tx.QueryRow(query)
	err := row.Scan(&pgChat.Chat, &pgChat.Author, &pgChat.ReadOnly, &pgChat.SentAt)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoOutboxChats
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, ErrInternal
	}
	return &domain.Chat{Uuid: pgChat.Chat, Owner: domain.User{Uuid: pgChat.Author}, Readonly: pgChat.ReadOnly}, nil
}

func (p *Postgres) ConfirmOutboxChatSended(ctx context.Context, chatUuid uuid.UUID) error {
	const op = "postgres.ConfirmOutboxChatSended"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := fmt.Sprintf(`UPDATE %s SET sent_to_kafka = $1 WHERE chat = $2`, outboxChatsTable)
	_, err := tx.Exec(query, time.Now(), chatUuid.String())
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return ErrInternal
	}
	return nil
}

func (p *Postgres) CreateMessageOutbox(ctx context.Context, message domain.Message) error {
	const op = "postgres.CreateMessageOutbox"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := fmt.Sprintf("INSERT INTO %s (author, body, published) VALUES ($1,$2,$3)", outboxMessagesTable)
	_, err := tx.Exec(query, message.AuthorUuid, message.Body, message.Published)
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return ErrInternal
	}

	return nil
}

func (p *Postgres) GetNextOutboxMessage(ctx context.Context) (*domain.Message, error) {
	const op = "postgres.GetNextOutboxMessage"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	msg := domain.Message{}

	query := fmt.Sprintf("SELECT id, author, body, published FROM %s WHERE sent_to_kafka IS NULL LIMIT 1", outboxMessagesTable)
	row := tx.QueryRow(query)
	err := row.Scan(&msg.Id, &msg.AuthorUuid, &msg.Body, &msg.Published)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoOutboxMessages
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, ErrInternal
	}
	return &msg, nil
}

func (p *Postgres) ConfirmOutboxMessageSended(ctx context.Context, messageId int) error {
	const op = "postgres.ConfirmOutboxMessageSended"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := fmt.Sprintf(`UPDATE %s SET sent_to_kafka = $1 WHERE id = $2`, outboxMessagesTable)
	_, err := tx.Exec(query, time.Now(), messageId)
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return ErrInternal
	}
	return nil
}
