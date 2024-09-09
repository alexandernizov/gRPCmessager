package auth

import (
	"context"
	"errors"
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
	WithTx(ctx context.Context, tFunc func(ctx context.Context) error) error

	CreateUser(ctx context.Context, user domain.User) (*domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (*domain.User, error)
	GetUserByUuid(ctx context.Context, uuid uuid.UUID) (*domain.User, error)

	UpsertRefreshToken(ctx context.Context, userUuid uuid.UUID, refreshToken string) error
	GetRefreshToken(ctx context.Context, userUuid uuid.UUID) (string, error)
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
	ErrInternalError      = errors.New("internal error")
)

func New(log *slog.Logger, authStorage AuthStorage, jwtParams JwtParams) *AuthService {
	return &AuthService{log: log, authStorage: authStorage, jwtParams: jwtParams}
}

func (a *AuthService) Register(ctx context.Context, login, password string) (*domain.User, error) {
	const op = "auth.Register"
	log := a.log.With(slog.String("op", op))

	// Generate user
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return nil, ErrInvalidCredentials
	}

	newUser := domain.User{Uuid: uuid.New(), Login: login, PasswordHash: passHash}

	fn := func(fnCtx context.Context) error {
		// Check if user already exists
		_, err := a.authStorage.GetUserByLogin(fnCtx, newUser.Login)

		switch {
		case errors.Is(err, postgres.ErrUserNotFound):
		case err == nil:
			return ErrUserAlreadyExsist
		default:
			return ErrInternalError
		}

		// If user doesn't exist then create
		_, err = a.authStorage.CreateUser(fnCtx, newUser)
		if err != nil {
			return ErrInternalError
		}
		return nil
	}

	err = a.authStorage.WithTx(ctx, fn)
	if err != nil {
		return nil, err
	}

	return &newUser, nil
}

func (a *AuthService) Login(ctx context.Context, login, password string) (*domain.Tokens, error) {
	const op = "auth.Login"
	log := a.log.With(slog.String("op", op))

	var tokens domain.Tokens

	fn := func(fnCtx context.Context) error {
		user, err := a.authStorage.GetUserByLogin(fnCtx, login)
		if errors.Is(err, postgres.ErrUserNotFound) {
			return ErrInvalidCredentials
		}
		if err != nil {
			return ErrInternalError
		}

		if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
			log.Info("attempting to login with incorrect password", slog.String("userUuid", user.Uuid.String()))
			return ErrInvalidCredentials
		}

		tokens, err = jwt.NewTokens(*user, a.jwtParams.AccessTtl, a.jwtParams.RefreshTtl, a.jwtParams.Secret)
		if err != nil {
			log.Error("error with generating tokens", sl.Err(err))
			return ErrInternalError
		}

		err = a.authStorage.UpsertRefreshToken(fnCtx, user.Uuid, tokens.RefreshToken)
		if err != nil {
			return ErrInternalError
		}
		return nil
	}

	err := a.authStorage.WithTx(ctx, fn)
	if err != nil {
		return nil, err
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

	userUuid, err := jwt.GetUserUuidFromToken(token, a.jwtParams.Secret)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	var newTokens domain.Tokens
	fn := func(fnCtx context.Context) error {
		currentToken, err := a.authStorage.GetRefreshToken(fnCtx, userUuid)
		if err != nil {
			return ErrInvalidCredentials
		}

		if currentToken != token {
			log.Warn("attempting refresh tokens with expired token: ", slog.String("user_uuid", userUuid.String()))
			return ErrInvalidCredentials
		}

		user, err := a.authStorage.GetUserByUuid(fnCtx, userUuid)
		if err != nil {
			return ErrInvalidCredentials
		}

		// Generate new token
		newTokens, err = jwt.NewTokens(*user, a.jwtParams.AccessTtl, a.jwtParams.RefreshTtl, a.jwtParams.Secret)
		if err != nil {
			log.Error("error according creating jwt tokens: ", sl.Err(err))
			return ErrInternalError
		}

		err = a.authStorage.UpsertRefreshToken(fnCtx, userUuid, newTokens.RefreshToken)
		if err != nil {
			return ErrInternalError
		}

		return nil
	}

	err = a.authStorage.WithTx(ctx, fn)
	if err != nil {
		return nil, err
	}

	return &newTokens, nil
}
