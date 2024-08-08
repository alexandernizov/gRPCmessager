package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	repo := postgres.New(log, db)

	type mockBehavior func(func(context.Context) error)

	sameError := errors.New("some error")

	testTable := []struct {
		name         string
		ctx          context.Context
		testFunc     func(context.Context) error
		expectErr    error
		mockBehavior mockBehavior
	}{
		{
			name:      "transaction_commited",
			ctx:       context.Background(),
			testFunc:  func(ctx context.Context) error { return nil },
			expectErr: nil,
			mockBehavior: func(func(context.Context) error) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
		},
		{
			name:      "begin_errored",
			ctx:       context.Background(),
			testFunc:  func(ctx context.Context) error { return nil },
			expectErr: postgres.ErrInternal,
			mockBehavior: func(func(context.Context) error) {
				mock.ExpectBegin().WillReturnError(errors.New("some error"))
			},
		},
		{
			name:      "fn_errored",
			ctx:       context.Background(),
			testFunc:  func(ctx context.Context) error { return sameError },
			expectErr: sameError,
			mockBehavior: func(func(context.Context) error) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
		},
		{
			name:      "fn_errored_rollback_errored",
			ctx:       context.Background(),
			testFunc:  func(ctx context.Context) error { return sameError },
			expectErr: postgres.ErrInternal,
			mockBehavior: func(func(context.Context) error) {
				mock.ExpectBegin()
				mock.ExpectRollback().WillReturnError(errors.New("some error"))
			},
		},
		{
			name:      "fn_commit_errored",
			ctx:       context.Background(),
			testFunc:  func(ctx context.Context) error { return nil },
			expectErr: postgres.ErrInternal,
			mockBehavior: func(func(context.Context) error) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(errors.New("some error"))
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockBehavior(testCase.testFunc)

			err := repo.WithTx(testCase.ctx, testCase.testFunc)

			assert.Equal(t, testCase.expectErr, err)

			mock.ExpectationsWereMet()
		})
	}
}

func TestCreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	repo := postgres.New(log, db)

	type args struct {
		context.Context
		domain.User
	}

	type mockBehavior func(args)

	sameUuid := uuid.New()

	testTable := []struct {
		name         string
		args         args
		expectUser   *domain.User
		expectErr    error
		mockBehavior mockBehavior
	}{
		{
			name: "user_created",
			args: args{
				context.Background(),
				domain.User{Uuid: sameUuid, Login: "test", PasswordHash: []byte("test")},
			},
			expectUser: &domain.User{Uuid: sameUuid, Login: "test", PasswordHash: []byte("test")},
			expectErr:  nil,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO users").WithArgs(args.Uuid, args.Login, args.PasswordHash).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "got_internal_error",
			args: args{
				context.Background(),
				domain.User{Uuid: sameUuid, Login: "test", PasswordHash: []byte("test")},
			},
			expectUser: nil,
			expectErr:  postgres.ErrInternal,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO users").WithArgs(args.Uuid, args.Login, args.PasswordHash).
					WillReturnError(errors.New("some error"))
				mock.ExpectRollback()
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockBehavior(testCase.args)

			user, err := repo.CreateUser(testCase.args.Context, testCase.args.User)

			assert.Equal(t, testCase.expectUser, user)
			assert.Equal(t, testCase.expectErr, err)

			mock.ExpectationsWereMet()
		})
	}
}

func TestGetUserByLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	repo := postgres.New(log, db)

	type args struct {
		ctx   context.Context
		login string
	}

	type mockBehavior func(args)

	sameUuid := uuid.New()

	testTable := []struct {
		name         string
		args         args
		expectUser   *domain.User
		expectErr    error
		mockBehavior mockBehavior
	}{
		{
			name: "got_user",
			args: args{
				ctx:   context.Background(),
				login: "test",
			},
			expectUser: &domain.User{Uuid: sameUuid, Login: "test", PasswordHash: []byte("test")},
			expectErr:  nil,
			mockBehavior: func(args args) {
				rows := sqlmock.NewRows([]string{"uuid", "login", "password"}).
					AddRow(sameUuid, "test", "test")

				mock.ExpectBegin()
				mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.login = ?").WithArgs(args.login).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
		},
		{
			name: "got_error_user_not_found",
			args: args{
				ctx:   context.Background(),
				login: "test",
			},
			expectUser: nil,
			expectErr:  postgres.ErrUserNotFound,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.login = ?").WithArgs(args.login).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
		},
		{
			name: "got_internal_error",
			args: args{
				ctx:   context.Background(),
				login: "test",
			},
			expectUser: nil,
			expectErr:  postgres.ErrInternal,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.login = ?").WithArgs(args.login).
					WillReturnError(errors.New("some error"))
				mock.ExpectRollback()
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockBehavior(testCase.args)

			user, err := repo.GetUserByLogin(testCase.args.ctx, testCase.args.login)

			assert.Equal(t, testCase.expectUser, user)
			assert.Equal(t, testCase.expectErr, err)

			mock.ExpectationsWereMet()
		})
	}
}

func TestGetUserByUuid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	repo := postgres.New(log, db)

	type args struct {
		ctx  context.Context
		uuid uuid.UUID
	}

	type mockBehavior func(args)

	sameUuid := uuid.New()

	testTable := []struct {
		name         string
		args         args
		expectUser   *domain.User
		expectErr    error
		mockBehavior mockBehavior
	}{
		{
			name: "got_user",
			args: args{
				ctx:  context.Background(),
				uuid: sameUuid,
			},
			expectUser: &domain.User{Uuid: sameUuid, Login: "test", PasswordHash: []byte("test")},
			expectErr:  nil,
			mockBehavior: func(args args) {
				rows := sqlmock.NewRows([]string{"uuid", "login", "password"}).
					AddRow(sameUuid, "test", "test")

				mock.ExpectBegin()
				mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.uuid = ?").WithArgs(args.uuid).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
		},
		{
			name: "got_error_user_not_found",
			args: args{
				ctx:  context.Background(),
				uuid: sameUuid,
			},
			expectUser: nil,
			expectErr:  postgres.ErrUserNotFound,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.uuid = ?").WithArgs(args.uuid).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
		},
		{
			name: "got_internal_error",
			args: args{
				ctx:  context.Background(),
				uuid: sameUuid,
			},
			expectUser: nil,
			expectErr:  postgres.ErrInternal,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT uuid, login, password FROM users WHERE users.uuid = ?").WithArgs(args.uuid).
					WillReturnError(errors.New("some error"))
				mock.ExpectRollback()
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockBehavior(testCase.args)

			user, err := repo.GetUserByUuid(testCase.args.ctx, testCase.args.uuid)

			assert.Equal(t, testCase.expectUser, user)
			assert.Equal(t, testCase.expectErr, err)

			mock.ExpectationsWereMet()
		})
	}
}

func TestUpsertRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	repo := postgres.New(log, db)

	type args struct {
		ctx          context.Context
		userUuid     uuid.UUID
		refreshToken string
	}

	type mockBehavior func(args)

	sameUuid := uuid.New()

	testTable := []struct {
		name         string
		args         args
		expectErr    error
		mockBehavior mockBehavior
	}{
		{
			name: "refresh_token_inserted",
			args: args{
				ctx:          context.Background(),
				userUuid:     sameUuid,
				refreshToken: "test",
			},
			expectErr: nil,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO refresh_tokens").
					WithArgs(args.userUuid, args.refreshToken).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "got_internal_error",
			args: args{
				ctx:          context.Background(),
				userUuid:     sameUuid,
				refreshToken: "test",
			},
			expectErr: postgres.ErrInternal,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO refresh_tokens").
					WithArgs(args.userUuid, args.refreshToken).
					WillReturnError(errors.New("some error"))
				mock.ExpectRollback()
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockBehavior(testCase.args)

			err := repo.UpsertRefreshToken(testCase.args.ctx, testCase.args.userUuid, testCase.args.refreshToken)

			assert.Equal(t, testCase.expectErr, err)

			mock.ExpectationsWereMet()
		})
	}
}

func TestGetRefreshToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	repo := postgres.New(log, db)

	type args struct {
		ctx      context.Context
		userUuid uuid.UUID
	}

	type mockBehavior func(args)

	sameUuid := uuid.New()

	testTable := []struct {
		name         string
		args         args
		expectToken  string
		expectErr    error
		mockBehavior mockBehavior
	}{
		{
			name: "got_refresh_token",
			args: args{
				ctx:      context.Background(),
				userUuid: sameUuid,
			},
			expectToken: "test",
			expectErr:   nil,
			mockBehavior: func(args args) {
				rows := sqlmock.NewRows([]string{"token"}).
					AddRow("test")

				mock.ExpectBegin()
				mock.ExpectQuery("SELECT token FROM refresh_tokens WHERE user_uuid = ?").
					WithArgs(args.userUuid).
					WillReturnRows(rows)
				mock.ExpectCommit()
			},
		},
		{
			name: "got_error_no_rows",
			args: args{
				ctx:      context.Background(),
				userUuid: sameUuid,
			},
			expectToken: "",
			expectErr:   postgres.ErrTokenNotFound,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT token FROM refresh_tokens WHERE user_uuid = ?").
					WithArgs(args.userUuid).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
		},
		{
			name: "got_error_internal",
			args: args{
				ctx:      context.Background(),
				userUuid: sameUuid,
			},
			expectToken: "",
			expectErr:   postgres.ErrInternal,
			mockBehavior: func(args args) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT token FROM refresh_tokens WHERE user_uuid = ?").
					WithArgs(args.userUuid).
					WillReturnError(errors.New("some error"))
				mock.ExpectRollback()
			},
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.mockBehavior(testCase.args)

			token, err := repo.GetRefreshToken(testCase.args.ctx, testCase.args.userUuid)

			assert.Equal(t, testCase.expectErr, err)
			assert.Equal(t, testCase.expectToken, token)

			mock.ExpectationsWereMet()
		})
	}
}
