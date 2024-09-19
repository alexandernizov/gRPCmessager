package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/jwt"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"github.com/alexandernizov/grpcmessanger/internal/storage"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthStorage interface {
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
	_, err = a.authStorage.GetUserByLogin(ctx, newUser.Login)

	switch {
	case errors.Is(err, storage.ErrUserNotFound):
	case err == nil:
		return nil, ErrUserAlreadyExsist
	default:
		return nil, ErrInternalError
	}

	// If user doesn't exist then create
	_, err = a.authStorage.CreateUser(ctx, newUser)
	if err != nil {
		return nil, ErrInternalError
	}
	return &newUser, nil
}

func (a *AuthService) Login(ctx context.Context, login, password string) (*domain.Tokens, error) {
	const op = "auth.Login"
	log := a.log.With(slog.String("op", op))

	var tokens domain.Tokens

	user, err := a.authStorage.GetUserByLogin(ctx, login)
	if errors.Is(err, storage.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, ErrInternalError
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		log.Info("attempting to login with incorrect password", slog.String("userUuid", user.Uuid.String()))
		return nil, ErrInvalidCredentials
	}

	tokens, err = jwt.NewTokens(*user, a.jwtParams.AccessTtl, a.jwtParams.RefreshTtl, a.jwtParams.Secret)
	if err != nil {
		log.Error("error with generating tokens", sl.Err(err))
		return nil, ErrInternalError
	}

	err = a.authStorage.UpsertRefreshToken(ctx, user.Uuid, tokens.RefreshToken)
	if err != nil {
		return nil, ErrInternalError
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
	currentToken, err := a.authStorage.GetRefreshToken(ctx, userUuid)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if currentToken != token {
		log.Warn("attempting refresh tokens with expired token: ", slog.String("user_uuid", userUuid.String()))
		return nil, ErrInvalidCredentials
	}

	user, err := a.authStorage.GetUserByUuid(ctx, userUuid)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate new token
	newTokens, err = jwt.NewTokens(*user, a.jwtParams.AccessTtl, a.jwtParams.RefreshTtl, a.jwtParams.Secret)
	if err != nil {
		log.Error("error according creating jwt tokens: ", sl.Err(err))
		return nil, ErrInternalError
	}

	err = a.authStorage.UpsertRefreshToken(ctx, userUuid, newTokens.RefreshToken)
	if err != nil {
		return nil, ErrInternalError
	}

	return &newTokens, nil
}
