package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/lib/jwt"
	"github.com/alexandernizov/grpcmessanger/internal/lib/logger/sl"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthStorage interface {
	NewUser(ctx context.Context, user domain.User) (bool, error)
	GetUser(ctx context.Context, login string) (domain.User, error)
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

func NewAuthService(log *slog.Logger, authStorage AuthStorage, ttl time.Duration, secret []byte) *AuthService {
	return &AuthService{log: log, authStorage: authStorage, jwtParams: JwtParams{ttl, secret}}
}

func (a *AuthService) Register(ctx context.Context, login, password string) (bool, error) {
	const op = "auth.Register"
	log := a.log.With(slog.String("op", op))

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return false, fmt.Errorf("%s, %w", op, err)
	}

	var user domain.User
	user.Uuid = uuid.New()
	user.Login = login
	user.PasswordHash = passHash

	result, err := a.authStorage.NewUser(ctx, user)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}

func (a *AuthService) Login(ctx context.Context, login, password string) (string, error) {
	const op = "auth.Register"
	log := a.log.With(slog.String("op", op))

	user, err := a.authStorage.GetUser(ctx, login)
	if err != nil {
		return "", err
	}
	if user.Uuid == uuid.Nil {
		return "", errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		log.Info("invalid credentials", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, errors.New("invalid credentionals"))
	}

	token, err := jwt.NewToken(user, a.jwtParams.ttl, a.jwtParams.secret)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (a *AuthService) Validate(ctx context.Context, token string) (bool, error) {
	result, err := jwt.ValidateToken(token, a.jwtParams.secret)
	return result, err
}
