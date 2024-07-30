package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/alexandernizov/grpcmessanger/internal/domain"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	_ "github.com/lib/pq"
)

type Postgres struct {
	log *slog.Logger
	db  *sql.DB
}

func NewPostgresAuthStorage(log *slog.Logger, host, port, user, password, dbname, migrationPath string) (*Postgres, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("could not open db connection: %v", err)
	}

	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", user, password, host, port)
	err = initMigrate(migrationPath, connectionString)
	if err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not connect to db: %w", err)
	}

	return &Postgres{log: log, db: db}, nil
}

func initMigrate(migrationsPath string, connectionString string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		connectionString,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (p *Postgres) Close() {
	p.db.Close()
}

func (p *Postgres) NewUser(ctx context.Context, user domain.User) (bool, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return false, fmt.Errorf("could not insert user: %w", err)
	}

	query := "SELECT u.uuid FROM users as u WHERE u.login = $1"
	row := tx.QueryRow(query, user.Login)

	var login string
	row.Scan(&login)
	if len(login) > 0 {
		tx.Rollback()
		return false, fmt.Errorf("could not insert user: %w", errors.New("user already exists"))
	}

	query = `INSERT INTO users (uuid, login, password) VALUES ($1,$2,$3)`
	_, err = tx.Exec(query, user.Uuid, user.Login, user.PasswordHash)

	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("could not insert user: %w", err)
	}
	tx.Commit()
	return true, nil
}

func (p *Postgres) GetUser(ctx context.Context, login string) (domain.User, error) {
	var user domain.User
	query := "SELECT u.uuid, u.login, u.password FROM users as u WHERE u.login = $1"
	row := p.db.QueryRow(query, login)
	err := row.Scan(&user.Uuid, &user.Login, &user.PasswordHash)
	if err != nil {
		return user, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}
