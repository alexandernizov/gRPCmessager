package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/domain/errs"
	"github.com/google/uuid"
)

type AuthMemStorage struct {
	Users    []domain.User
	sessions sessionsStorage
}

func NewAuthMemStorage(sessionTTL time.Duration) *AuthMemStorage {
	mux := sync.RWMutex{}
	sessionsMap := make(map[uuid.UUID]domain.Session)
	sessionsStorage := sessionsStorage{mux: &mux, ttl: sessionTTL, sessions: sessionsMap}
	return &AuthMemStorage{sessions: sessionsStorage}
}

type sessionsStorage struct {
	mux      *sync.RWMutex
	ttl      time.Duration
	sessions map[uuid.UUID]domain.Session
}

func (a *AuthMemStorage) SaveUser(ctx context.Context, name string, password string) (bool, error) {
	// Check if user already exists
	_, err := a.GetUser(ctx, name, password)
	if !errors.Is(err, errs.ErrUserNotFound) {
		return false, errs.ErrUserAlreadyExists
	}
	// Register new user
	user := domain.User{Name: name, Password: password, Uuid: uuid.New()}
	a.Users = append(a.Users, user)
	// Add a new user
	return true, nil
}

func (a *AuthMemStorage) GetUser(ctx context.Context, name string, password string) (*domain.User, error) {
	for _, user := range a.Users {
		if user.Name == name && user.Password == password {
			return &user, nil
		}
	}
	return nil, errs.ErrUserNotFound
}

func (a *AuthMemStorage) MakeUserSession(ctx context.Context, user *domain.User) (uuid.UUID, error) {
	newSession := domain.Session{SessionUUID: uuid.New(), UserUUID: user.Uuid, ExpiresAt: time.Now().Add(a.sessions.ttl)}
	a.sessions.mux.Lock()
	a.sessions.sessions[user.Uuid] = newSession
	a.sessions.mux.Unlock()
	return newSession.SessionUUID, nil
}

func (a *AuthMemStorage) GetSession(ctx context.Context, user *domain.User) (*domain.Session, error) {
	a.sessions.mux.RLock()
	defer a.sessions.mux.RUnlock()
	session, ok := a.sessions.sessions[user.Uuid]
	if !ok {
		return nil, errs.ErrSessionExpired
	}
	return &session, nil
}
