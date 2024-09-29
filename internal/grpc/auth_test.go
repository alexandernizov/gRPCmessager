package grpc

import (
	"context"
	"reflect"
	"testing"

	"github.com/alexandernizov/grpcmessanger/api/gen/authpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/grpc/mocks"
	"github.com/alexandernizov/grpcmessanger/internal/services/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestAuthServer_Register(t *testing.T) {
	type mockArgs struct {
		methodName string
		arguments  []any
		returning  []any
	}
	type funcArgs struct {
		ctx context.Context
		req *authpb.RegisterReq
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs mockArgs
		want     *authpb.RegisterResp
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &authpb.RegisterReq{Login: "Test", Password: "Test"},
			},
			mockArgs: mockArgs{methodName: "Register", arguments: []any{mock.Anything, "Test", "Test"}, returning: []any{&domain.User{Uuid: uuid.New()}, nil}},
			want:     &authpb.RegisterResp{Registred: true},
			wantErr:  false,
		},
		{
			name: "empty_login",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &authpb.RegisterReq{Login: "", Password: "Test"},
			},
			mockArgs: mockArgs{methodName: ""},
			want:     nil,
			wantErr:  true,
		},
		{
			name: "empty_password",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &authpb.RegisterReq{Login: "Test", Password: ""},
			},
			mockArgs: mockArgs{methodName: ""},
			want:     nil,
			wantErr:  true,
		},
		{
			name: "already_exists",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &authpb.RegisterReq{Login: "Test", Password: "Test"},
			},
			mockArgs: mockArgs{methodName: "Register", arguments: []any{mock.Anything, "Test", "Test"}, returning: []any{nil, auth.ErrUserAlreadyExsist}},
			want:     nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authProvider := mocks.NewAuthProvider(t)
			if tt.mockArgs.methodName > "" {
				authProvider.On(tt.mockArgs.methodName, tt.mockArgs.arguments...).Return(tt.mockArgs.returning...).Once()
			}
			a := &AuthServer{
				Provider: authProvider,
			}
			got, err := a.Register(tt.funcArgs.ctx, tt.funcArgs.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthServer.Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthServer.Register() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthServer_Login(t *testing.T) {
	type mockArgs struct {
		methodName string
		arguments  []any
		returning  []any
	}
	type funcArgs struct {
		ctx context.Context
		req *authpb.LoginReq
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs mockArgs
		want     *authpb.LoginResp
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &authpb.LoginReq{Login: "Test", Password: "Test"},
			},
			mockArgs: mockArgs{methodName: "Login", arguments: []any{mock.Anything, "Test", "Test"}, returning: []any{&domain.Tokens{AccessToken: "test", RefreshToken: "test"}, nil}},
			want:     &authpb.LoginResp{AccessToken: "test", RefreshToken: "test"},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authProvider := mocks.NewAuthProvider(t)
			if tt.mockArgs.methodName > "" {
				authProvider.On(tt.mockArgs.methodName, tt.mockArgs.arguments...).Return(tt.mockArgs.returning...).Once()
			}
			a := &AuthServer{
				Provider: authProvider,
			}
			got, err := a.Login(tt.funcArgs.ctx, tt.funcArgs.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthServer.Login() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthServer.Login() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthServer_Refresh(t *testing.T) {
	type mockArgs struct {
		methodName string
		arguments  []any
		returning  []any
	}
	type funcArgs struct {
		ctx context.Context
		req *authpb.RefreshReq
	}
	tests := []struct {
		name     string
		funcArgs funcArgs
		mockArgs mockArgs
		want     *authpb.RefreshResp
		wantErr  bool
	}{
		{
			name: "success",
			funcArgs: funcArgs{
				ctx: context.Background(),
				req: &authpb.RefreshReq{RefreshToken: "test"},
			},
			mockArgs: mockArgs{methodName: "Refresh", arguments: []any{mock.Anything, "test"}, returning: []any{&domain.Tokens{AccessToken: "test", RefreshToken: "test"}, nil}},
			want:     &authpb.RefreshResp{AccessToken: "test", RefreshToken: "test"},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authProvider := mocks.NewAuthProvider(t)
			if tt.mockArgs.methodName > "" {
				authProvider.On(tt.mockArgs.methodName, tt.mockArgs.arguments...).Return(tt.mockArgs.returning...).Once()
			}
			a := &AuthServer{
				Provider: authProvider,
			}
			got, err := a.Refresh(tt.funcArgs.ctx, tt.funcArgs.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthServer.Refresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthServer.Refresh() = %v, want %v", got, tt.want)
			}
		})
	}
}
