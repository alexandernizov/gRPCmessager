package auth

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/jwt"
	"github.com/alexandernizov/grpcmessanger/internal/services/auth/mocks"
	"github.com/alexandernizov/grpcmessanger/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

var (
	secretTest         = "test"
	userUuidTest       = uuid.MustParse("8ee4e645-b894-4477-820b-48381e10677f")
	hashedPasswordTest = "$2a$10$yEC5DhDM3Mx4f4Wex2qqZ..ZK1vh4a/Q25x4Zm/RWztFCgsUZvVBy"
	userTest           = domain.User{Uuid: userUuidTest, Login: "test", PasswordHash: []byte("test")}
	tokensTest, _      = jwt.NewTokens(userTest, time.Minute, time.Minute, []byte(secretTest))
)

type mockArgs struct {
	methodName string
	arguments  []any
	returning  []any
}

func NewMockService(t *testing.T, inputMocks []mockArgs) *AuthService {
	chatStorage := mocks.NewAuthStorage(t)
	for _, m := range inputMocks {
		chatStorage.On(m.methodName, m.arguments...).Return(m.returning...).Once()
	}
	mockService := AuthService{
		log: slog.Default(),

		authStorage: chatStorage,
		jwtParams: JwtParams{
			AccessTtl:  1 * time.Minute,
			RefreshTtl: 1 * time.Minute,
			Secret:     []byte(secretTest),
		},
	}
	return &mockService
}

func TestAuthService_Register(t *testing.T) {
	type funcArgs struct {
		ctx      context.Context
		login    string
		password string
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs []mockArgs
		want     *domain.User
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx:      context.TODO(),
				login:    "test",
				password: "test",
			},
			mockArgs: []mockArgs{
				{methodName: "GetUserByLogin", arguments: []any{mock.Anything, "test"}, returning: []any{nil, storage.ErrUserNotFound}},
				{methodName: "CreateUser", arguments: []any{mock.Anything, mock.Anything}, returning: []any{nil, nil}},
			},
			want:    &domain.User{Login: "test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewMockService(t, tt.mockArgs)
			got, err := a.Register(tt.funcArgs.ctx, tt.funcArgs.login, tt.funcArgs.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Login != tt.want.Login {
				t.Errorf("AuthService.Register() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	type funcArgs struct {
		ctx      context.Context
		login    string
		password string
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs []mockArgs
		want     *domain.Tokens
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx:      context.TODO(),
				login:    "test",
				password: "test",
			},
			mockArgs: []mockArgs{
				{methodName: "GetUserByLogin", arguments: []any{mock.Anything, "test"}, returning: []any{&domain.User{Uuid: userUuidTest, Login: "test", PasswordHash: []byte(hashedPasswordTest)}, nil}},
				{methodName: "UpsertRefreshToken", arguments: []any{mock.Anything, userUuidTest, mock.Anything}, returning: []any{nil}},
			},
			want:    &domain.Tokens{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewMockService(t, tt.mockArgs)
			got, err := a.Login(tt.funcArgs.ctx, tt.funcArgs.login, tt.funcArgs.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("AuthService.Login() = %v, want %v", got, tt.want)
			}
			if tt.want == nil && got != nil {
				t.Errorf("AuthService.Login() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	type funcArgs struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs []mockArgs
		want     *domain.Tokens
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx:   context.TODO(),
				token: tokensTest.RefreshToken,
			},
			mockArgs: []mockArgs{
				{methodName: "GetRefreshToken", arguments: []any{mock.Anything, userUuidTest}, returning: []any{tokensTest.RefreshToken, nil}},
				{methodName: "GetUserByUuid", arguments: []any{mock.Anything, userUuidTest}, returning: []any{&userTest, nil}},
				{methodName: "UpsertRefreshToken", arguments: []any{mock.Anything, userUuidTest, mock.Anything}, returning: []any{nil}},
			},
			want:    &domain.Tokens{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewMockService(t, tt.mockArgs)
			got, err := a.Refresh(tt.funcArgs.ctx, tt.funcArgs.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthService.Refresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("AuthService.Login() = %v, want %v", got, tt.want)
			}
			if tt.want == nil && got != nil {
				t.Errorf("AuthService.Login() = %v, want %v", got, tt.want)
			}
		})
	}
}
