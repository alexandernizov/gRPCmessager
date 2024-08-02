package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/jwt"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthStorage interface {
	WithinTransaction(ctx context.Context, tFunc func(ctx context.Context) error) error
	CreateUser(ctx context.Context, user domain.User) (*domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)
}

type AuthService struct {
	log *slog.Logger

	authStorage AuthStorage
	jwtParams   JwtParams
}

type JwtParams struct {
	ttl    time.Duration
	secret []byte
}

var (
	ErrUserAlreadyExsist  = errors.New("user is already exist")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrFailLogin          = errors.New("login failed")
)

func NewAuthService(log *slog.Logger, authStorage AuthStorage, ttl time.Duration, secret []byte) *AuthService {
	return &AuthService{log: log, authStorage: authStorage, jwtParams: JwtParams{ttl, secret}}
}

func (a *AuthService) Register(ctx context.Context, login, password string) (*domain.User, error) {
	const op = "auth.Register"
	log := a.log.With(slog.String("op", op))

	//1. Generate user
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return nil, fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	}

	user := domain.User{Uuid: uuid.New(), Login: login, PasswordHash: passHash}

	//2. Invoke in same transaction
	fn := func(fnCtx context.Context) error {
		//Check if user exists
		_, err := a.authStorage.GetUserByLogin(fnCtx, user.Login)
		if !errors.Is(err, postgres.ErrUserNotFound) {
			return ErrUserAlreadyExsist
		}
		//If user doesn't exist then create
		_, err = a.authStorage.CreateUser(fnCtx, user)
		if err != nil {
			log.Error("error according user creating", sl.Err(err))
			return err
		}
		return nil
	}

	err = a.authStorage.WithinTransaction(ctx, fn)
	if err != nil {
		return nil, err
	}
	//3. Return result
	return &user, nil
}

func (a *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	const op = "auth.Login"
	log := a.log.With(slog.String("op", op))

	user, err := a.authStorage.GetUserByLogin(ctx, login)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		log.Info("attempting to login with incorrect password")
		return "", fmt.Errorf("%w", ErrInvalidCredentials)
	}

	token, err := jwt.NewToken(*user, a.jwtParams.ttl, a.jwtParams.secret)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailLogin, err)
	}

	return token, nil
}

func (a *AuthService) Validate(ctx context.Context, token string) (bool, error) {
	return false, nil
	// result, err := jwt.ValidateToken(token, a.jwtParams.secret)
	// return result, err
}
