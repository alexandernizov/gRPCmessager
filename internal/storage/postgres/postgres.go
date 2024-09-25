package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/api/gen/outbox"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/proto"
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
	ErrNoConnection = errors.New("can't establish connection to db")
)

const (
	usersTable         = "users"
	refreshTokensTable = "refresh_tokens"
	chatsTable         = "chats"
	messagesTable      = "messages"
	outboxTable        = "outbox"
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
	Uuid     uuid.UUID  `pg:"uuid"`
	Owner    uuid.UUID  `pg:"owner"`
	ReadOnly bool       `pg:"read_only"`
	Deadline *time.Time `pg:"dead_line"`
}

type Message struct {
	Id         int        `pg:"id"`
	ChatUuid   uuid.UUID  `pg:"chat_uuid"`
	AuthorUuid uuid.UUID  `pg:"author_uuid"`
	Body       []byte     `pg:"body"`
	Published  *time.Time `pg:"dead_line"`
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
		return storage.ErrInternal
	}

	ctxTx := injectTx(ctx, tx)

	fnError := tFunc(ctxTx)

	if fnError != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Error("error with Rollback transaction", sl.Err(rollbackErr))
			return storage.ErrInternal
		}
		return fnError
	}

	if commitError := tx.Commit(); commitError != nil {
		log.Error("error with Commit transaction", sl.Err(commitError))
		return storage.ErrInternal
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
		return nil, storage.ErrInternal
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
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, storage.ErrInternal
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
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, storage.ErrInternal
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
		return storage.ErrInternal
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
		return "", storage.ErrTokenNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return "", storage.ErrInternal
	}

	return token, nil
}

func (p *Postgres) CreateChat(ctx context.Context, chat domain.Chat) (*domain.Chat, error) {
	const op = "postgres.CreateChat"
	log := p.log.With(slog.String("op", op))

	msg := outbox.OutboxChat{
		Uuid:      chat.Uuid.String(),
		OwnerUuid: chat.Owner.Uuid.String(),
		Readonly:  chat.Readonly,
		Deadline:  chat.Deadline.String(),
	}

	marshalledMessage, err := proto.Marshal(&msg)
	if err != nil {
		return &domain.Chat{}, storage.ErrInternal
	}

	tx, closeTx := p.extractTx(ctx)

	pgChat := Chat{Uuid: chat.Uuid, Owner: chat.Owner.Uuid, ReadOnly: chat.Readonly, Deadline: &chat.Deadline}

	query1 := fmt.Sprintf("INSERT INTO %s (uuid, owner, read_only, dead_line) VALUES ($1,$2,$3,$4)", chatsTable)
	query2 := fmt.Sprintf("INSERT INTO %s (uuid, topic, message) VALUES ($1,$2,$3)", outboxTable)
	_, err = tx.Exec(query1, pgChat.Uuid, pgChat.Owner, pgChat.ReadOnly, pgChat.Deadline)
	_, err = tx.Exec(query2, chat.Uuid, domain.ChatTopic, marshalledMessage)

	closeTx(err)

	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &chat, nil
}

func (p *Postgres) GetChat(ctx context.Context, chatUuid uuid.UUID) (*domain.Chat, error) {
	const op = "postgres.GetChat"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	var chat Chat

	query := fmt.Sprintf("SELECT uuid, owner, read_only, dead_line FROM %s WHERE uuid = $1;", chatsTable)
	row := tx.QueryRow(query, chatUuid)
	err := row.Scan(&chat.Uuid, &chat.Owner, &chat.ReadOnly, &chat.Deadline)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrTokenNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, storage.ErrInternal
	}

	user, err := p.GetUserByUuid(ctx, chat.Owner)
	if err != nil {
		return nil, storage.ErrInternal
	}

	return &domain.Chat{Uuid: chat.Uuid, Owner: *user, Readonly: chat.ReadOnly, Deadline: *chat.Deadline}, nil
}

func (p *Postgres) ChatsCount(ctx context.Context) (int, error) {
	const op = "postgres.ChatsCount"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	var count int
	query := "SELECT count (*) FROM chats;"
	row := tx.QueryRow(query)
	err := row.Scan(&count)
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return 0, storage.ErrInternal
	}

	return count, nil
}

func (p *Postgres) PostMessage(ctx context.Context, chat uuid.UUID, message domain.Message) (*domain.Message, error) {
	const op = "postgres.PostMessage"
	log := p.log.With(slog.String("op", op))

	msg := outbox.OutboxMessage{
		Id:         int64(message.Id),
		AuthorUuid: message.AuthorUuid.String(),
		Body:       message.Body,
		Published:  message.Published.String(),
	}

	marshalledMessage, err := proto.Marshal(&msg)
	if err != nil {
		return &domain.Message{}, storage.ErrInternal
	}

	tx, closeTx := p.extractTx(ctx)

	pgMessage := Message{ChatUuid: chat, AuthorUuid: message.AuthorUuid, Body: []byte(message.Body), Published: &message.Published}

	query1 := fmt.Sprintf("INSERT INTO %s (chat_uuid, author_uuid, body, published) VALUES ($1,$2,$3,$4)", messagesTable)
	query2 := fmt.Sprintf("INSERT INTO %s (uuid, topic, message) VALUES ($1,$2,$3)", outboxTable)

	_, err = tx.Exec(query1, pgMessage.ChatUuid, pgMessage.AuthorUuid, pgMessage.Body, pgMessage.Published)
	_, err = tx.Exec(query2, uuid.New(), domain.MessageTopic, marshalledMessage)
	closeTx(err)

	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &message, nil
}

func (p *Postgres) TrimMessages(ctx context.Context, chat uuid.UUID, maximumMessages int) (bool, error) {
	const op = "postgres.TrimMessages"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := `
	WITH numbered_messages AS (
    	SELECT 
        	id,
        	ROW_NUMBER() OVER (PARTITION BY chat_uuid ORDER BY published DESC) AS row_num
    	FROM 
        	messages
		WHERE
			chat_uuid = $1
	)
	DELETE FROM messages
	WHERE id IN (
    	SELECT id
    	FROM numbered_messages
    	WHERE row_num > $2
	);`

	_, err := tx.Exec(query, chat, maximumMessages)
	closeTx(err)
	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return false, storage.ErrInternal
	}

	return true, nil
}

func (p *Postgres) GetChatHistory(ctx context.Context, chatUuid uuid.UUID) ([]*domain.Message, error) {
	const op = "postgres.GetChatHistory"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	var res []*domain.Message

	query := fmt.Sprintf("SELECT id, author_uuid, body, published FROM %s WHERE chat_uuid = $1", messagesTable)
	rows, err := tx.Query(query, chatUuid)
	defer closeTx(err)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return res, nil
		}
		log.Error("error: ", sl.Err(err))
		return nil, err
	}

	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(&msg.Id, &msg.AuthorUuid, &msg.Body, &msg.Published)
		if err != nil {
			log.Error("error scanning row: ", sl.Err(err))
			return nil, err
		}
		res = append(res, &msg)
	}

	return res, nil
}

func (p *Postgres) GetNextOutbox(ctx context.Context) (*domain.Outbox, error) {
	const op = "postgres.GetNextOutbox"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	var next domain.Outbox

	query := fmt.Sprintf("SELECT uuid, topic, message FROM %s WHERE sent_at IS NULL;", outboxTable)
	row := tx.QueryRow(query)
	err := row.Scan(&next.Uuid, &next.Topic, &next.Message)
	closeTx(err)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNoOutbox
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return nil, storage.ErrInternal
	}

	return &next, nil
}

func (p *Postgres) ConfirmOutboxSended(ctx context.Context, outboxUuid uuid.UUID) error {
	const op = "postgres.ConfigOutboxSended"
	log := p.log.With(slog.String("op", op))

	tx, closeTx := p.extractTx(ctx)

	query := fmt.Sprintf("UPDATE %s SET sent_at = current_timestamp WHERE uuid = $1", outboxTable)
	_, err := tx.Exec(query, outboxUuid)
	closeTx(err)

	if err != nil {
		log.Info("error: ", sl.Err(err))
		return storage.ErrInternal
	}

	return nil
}
