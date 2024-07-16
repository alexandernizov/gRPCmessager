package auth

import (
	"context"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
)

type AuthService struct {
	authStorage AuthStorage
}

func NewAuthService(authStorage AuthStorage) *AuthService {
	return &AuthService{authStorage: authStorage}
}

type AuthStorage interface {
	SaveUser(ctx context.Context, name string, password string) (bool, error)
	GetUser(ctx context.Context, name string, password string) (*domain.User, error)
	MakeUserSession(ctx context.Context, user *domain.User) (uuid.UUID, error)
	GetSession(ctx context.Context, user *domain.User) (*domain.Session, error)
}

func (a *AuthService) Register(ctx context.Context, name, password string) (bool, error) {
	res, err := a.authStorage.SaveUser(ctx, name, password)
	return res, err
}

func (a *AuthService) Login(ctx context.Context, name, password string) (string, error) {
	user, err := a.authStorage.GetUser(ctx, name, password)
	if err != nil {
		return "", err
	}
	sessionUuid, err := a.authStorage.MakeUserSession(ctx, user)
	if err != nil {
		return "", err
	}
	return sessionUuid.String(), nil
}

func (a *AuthService) IsValid(ctx context.Context, name, password string) (bool, error) {
	user, err := a.authStorage.GetUser(ctx, name, password)
	if err != nil {
		return false, err
	}

	session, err := a.authStorage.GetSession(ctx, user)
	if err != nil {
		return false, err
	}
	if session.ExpiresAt.Before(time.Now()) {
		return false, nil
	}
	return true, nil
}

func (a *AuthService) GetUser(ctx context.Context, name, password string) (*domain.User, error) {
	return a.authStorage.GetUser(ctx, name, password)
}
