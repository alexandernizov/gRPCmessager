package postgres_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
)

func TestCreateUser(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	user := domain.User{
		Uuid:         uuid.New(),
		Login:        "testuser",
		PasswordHash: []byte("hash"),
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").WithArgs(user.Uuid, user.Login, user.PasswordHash).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	createdUser, err := pg.CreateUser(ctx, user)
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
}

func TestGetUserByLogin(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	login := "testuser"
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT uuid, login, password FROM users").WithArgs(login).
		WillReturnRows(sqlmock.NewRows([]string{"uuid", "login", "password"}).
			AddRow(uuid.New(), login, []byte("hash")))
	mock.ExpectCommit()

	ctx := context.Background()
	user, err := pg.GetUserByLogin(ctx, login)
	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestGetUserByUuid(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	userUuid := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.uuid = ?").WithArgs(userUuid).
		WillReturnRows(sqlmock.NewRows([]string{"uuid", "login", "password"}).
			AddRow(userUuid, "testuser", []byte("hash")))
	mock.ExpectCommit()

	ctx := context.Background()
	user, err := pg.GetUserByUuid(ctx, userUuid)
	assert.NoError(t, err)
	assert.NotNil(t, user)
}

func TestUpsertRefreshToken(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	userUuid := uuid.New()
	refreshToken := "some_refresh_token"

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO refresh_tokens").WithArgs(userUuid, refreshToken).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = pg.UpsertRefreshToken(ctx, userUuid, refreshToken)
	assert.NoError(t, err)
}

func TestGetRefreshToken(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	userUuid := uuid.New()
	expectedToken := "refresh_token"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT token FROM refresh_tokens WHERE user_uuid = ?").WithArgs(userUuid).
		WillReturnRows(sqlmock.NewRows([]string{"token"}).AddRow(expectedToken))
	mock.ExpectCommit()

	ctx := context.Background()
	token, err := pg.GetRefreshToken(ctx, userUuid)
	assert.NoError(t, err)
	assert.Equal(t, expectedToken, token)
}

func TestCreateChat(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	chat := domain.Chat{
		Uuid:     uuid.New(),
		Owner:    domain.User{Uuid: uuid.New()},
		Readonly: false,
		Deadline: time.Now(),
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO chats").WithArgs(chat.Uuid, chat.Owner.Uuid, chat.Readonly, chat.Deadline).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO outbox").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	createdChat, err := pg.CreateChat(ctx, chat)
	assert.NoError(t, err)
	assert.NotNil(t, createdChat)
}

func TestGetChat(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	chatUuid := uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT uuid, owner, read_only, dead_line FROM chats WHERE uuid = ?").WithArgs(chatUuid).
		WillReturnRows(sqlmock.NewRows([]string{"uuid", "owner", "read_only", "dead_line"}).
			AddRow(chatUuid, uuid.New(), false, time.Now()))
	mock.ExpectCommit()

	ctx := context.Background()
	chat, err := pg.GetChat(ctx, chatUuid)
	assert.NoError(t, err)
	assert.NotNil(t, chat)
}

func TestChatsCount(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT count (*) FROM chats").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectCommit()

	ctx := context.Background()
	count, err := pg.ChatsCount(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestPostMessage(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	chatUuid := uuid.New()
	messageUuid := uuid.New()
	message := domain.Message{
		Id:         1,
		AuthorUuid: messageUuid,
		Body:       "test message",
		Published:  time.Now(),
	}

	mock.ExpectBegin()

	// Ожидание вызова на вставку в таблицу outbox
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "Message", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	// Ожидание вызова на вставку в таблицу messages
	mock.ExpectExec("INSERT INTO messages").
		WithArgs(chatUuid, message.AuthorUuid, message.Body, message.Published).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	postedMessage, err := pg.PostMessage(ctx, chatUuid, message)
	assert.NoError(t, err)
	assert.NotNil(t, postedMessage)

	// Проверяем, что все ожидания были выполнены
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestTrimMessages(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	chatUuid := uuid.New()
	maximumMessages := 10

	mock.ExpectBegin()
	mock.ExpectExec("WITH numbered_messages AS").WithArgs(chatUuid, maximumMessages).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	result, err := pg.TrimMessages(ctx, chatUuid, maximumMessages)
	assert.NoError(t, err)
	assert.True(t, result)
}

func TestConfirmOutboxSended(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pg := postgres.New(log, db)

	outboxUuid := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE outbox SET sent_at").WithArgs(outboxUuid).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ctx := context.Background()
	err = pg.ConfirmOutboxSended(ctx, outboxUuid)
	assert.NoError(t, err)
}
