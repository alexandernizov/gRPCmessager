package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Postgres struct {
	log *slog.Logger
	db  *sql.DB
}

type PostgresOptions struct {
	Host     string
	Port     string
	User     string
	Password string
	DBname   string
}

var (
	ErrNoConnection = errors.New("can't establish connection to db")
	ErrBeginTx      = errors.New("can't begin transaction")
	ErrCommitTx     = errors.New("can't commit transaction")
	ErrRollbackTx   = errors.New("can't rollback transaction")
	ErrTxExec       = errors.New("can't execute transaction")
	ErrUserNotFound = errors.New("user is not found")
)

func New(log *slog.Logger, opt PostgresOptions) (*Postgres, error) {
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

type txKey struct{}

func injectTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func extractTx(ctx context.Context) *sql.Tx {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if !ok {
		return nil
	}
	return tx
}

func (p *Postgres) getTx(ctx context.Context) (*sql.Tx, error) {
	tx := extractTx(ctx)
	if tx == nil {
		return p.db.Begin()
	}
	return tx, nil
}

func (p *Postgres) WithinTransaction(ctx context.Context, tFunc func(ctx context.Context) error) error {
	tx, err := p.db.Begin()
	if err != nil {
		p.log.Error("begin transaction: %v", sl.Err(err))
		return fmt.Errorf("%w: %w", ErrBeginTx, err)
	}

	funcCtx := injectTx(ctx, tx)
	err = tFunc(funcCtx)

	if err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			p.log.Error("rollback transaction: %v", sl.Err(errRollback))
			return fmt.Errorf("%w: %w", ErrRollbackTx, errRollback)
		}
		return err
	}

	if errCommit := tx.Commit(); errCommit != nil {
		p.log.Error("commit transaction: %v", sl.Err(errCommit))
		return fmt.Errorf("%w: %w", ErrBeginTx, err)
	}
	return nil
}

type User struct {
	Uuid         uuid.UUID `pg:"uuid"`
	Login        string    `pg:"login"`
	PasswordHash []byte    `pg:"password"`
}

func (p *Postgres) CreateUser(ctx context.Context, user domain.User) (*domain.User, error) {
	const op = "postgres.CreateUser"
	log := p.log.With(slog.String("op", op))

	pgUser := User{Uuid: user.Uuid, Login: user.Login, PasswordHash: user.PasswordHash}
	query := "INSERT INTO users (uuid, login, password) VALUES ($1,$2,$3)"

	tx, err := p.getTx(ctx)
	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return nil, ErrBeginTx
	}

	_, err = tx.Exec(query, pgUser.Uuid, pgUser.Login, pgUser.PasswordHash)
	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return nil, ErrTxExec
	}

	return &user, nil
}

func (p *Postgres) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	const op = "postgres.GetUserByLogin"
	log := p.log.With(slog.String("op", op))

	pgUser := User{Login: login}

	query := "SELECT uuid, login, password FROM users WHERE users.login = $1"

	tx, err := p.getTx(ctx)
	if err != nil {
		log.Error("error: %v", sl.Err(err))
		return nil, ErrBeginTx
	}

	row := tx.QueryRow(query, pgUser.Login)
	err = row.Scan(&pgUser.Uuid, &pgUser.Login, &pgUser.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return &domain.User{}, ErrUserNotFound
	}
	if err != nil {
		log.Info("error: ", sl.Err(err))
		return &domain.User{}, fmt.Errorf("%w: %w", ErrTxExec, err)
	}

	return &domain.User{Uuid: pgUser.Uuid, Login: pgUser.Login, PasswordHash: pgUser.PasswordHash}, nil
}
