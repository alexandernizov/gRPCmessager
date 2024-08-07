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
	GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*domain.User, error)

	UpdateToken(ctx context.Context, userUuid uuid.UUID, refreshToken string) error
	GetUserToken(ctx context.Context, userUuid uuid.UUID) (string, error)
}

type AuthService struct {
	log *slog.Logger

	authStorage AuthStorage
	jwtParams   JwtParams
}

type JwtParams struct {
	AccessTtl  time.Duration
	RefreshTtl time.Duration
	Secret     []byte
}

var (
	ErrUserAlreadyExsist  = errors.New("user is already exist")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrFailLogin          = errors.New("login failed")
	ErrInvalidToken       = errors.New("token is invalid")
	ErrTokenIsExpired     = errors.New("refresh token is expired")
	ErrInternalError      = errors.New("internal error")
)

func NewAuthService(log *slog.Logger, authStorage AuthStorage, jwtParams JwtParams) *AuthService {
	return &AuthService{log: log, authStorage: authStorage, jwtParams: jwtParams}
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

func (a *AuthService) Login(ctx context.Context, login, password string) (*domain.Tokens, error) {
	const op = "auth.Login"
	log := a.log.With(slog.String("op", op))

	var tokens domain.Tokens

	fn := func(fnCtx context.Context) error {
		user, err := a.authStorage.GetUserByLogin(fnCtx, login)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidCredentials, err)
		}

		if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
			log.Info("attempting to login with incorrect password")
			return fmt.Errorf("%w", ErrInvalidCredentials)
		}

		tokens, err = jwt.NewTokens(*user, a.jwtParams.AccessTtl, a.jwtParams.RefreshTtl, a.jwtParams.Secret)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrFailLogin, err)
		}

		err = a.authStorage.UpdateToken(fnCtx, user.Uuid, tokens.RefreshToken)
		if err != nil {
			return err
		}
		return nil
	}

	err := a.authStorage.WithinTransaction(ctx, fn)
	if err != nil {
		return nil, ErrFailLogin
	}

	return &tokens, nil
}

func (a *AuthService) Refresh(ctx context.Context, token string) (*domain.Tokens, error) {
	const op = "auth.Refresh"
	log := a.log.With(slog.String("op", op))

	// Validate token
	ok, err := jwt.ValidateToken(token, a.jwtParams.Secret)
	if !ok || (err != nil) {
		log.Warn("someone send invalid token: ", sl.Err(err))
		return nil, ErrInvalidCredentials
	}

	// Get user
	userUuid, err := jwt.GetUserUuidFromToken(token, a.jwtParams.Secret)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := a.authStorage.GetUserByUuid(ctx, userUuid)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate new token
	newTokens, err := jwt.NewTokens(*user, a.jwtParams.AccessTtl, a.jwtParams.RefreshTtl, a.jwtParams.Secret)
	if err != nil {
		return nil, ErrInternalError
	}

	var currentToken string
	a.authStorage.WithinTransaction(ctx, func(fnCtx context.Context) error {
		currentToken, _ = a.authStorage.GetUserToken(fnCtx, userUuid)
		return nil
	})

	// Same transaction
	fn := func(fnCtx context.Context) error {
		// Get current refresh token in DB and compare it to provided token
		if err != nil {
			return ErrInvalidCredentials
		}

		if token != currentToken {
			log.Warn("someone trying to use expired token", slog.String("user_uuid", userUuid.String()))
			return ErrTokenIsExpired
		}

		err = a.authStorage.UpdateToken(ctx, userUuid, newTokens.RefreshToken)
		if err != nil {
			return ErrInternalError
		}
		return nil
	}

	err = a.authStorage.WithinTransaction(ctx, fn)
	if err != nil {
		return nil, err
	}

	return &newTokens, nil
}
