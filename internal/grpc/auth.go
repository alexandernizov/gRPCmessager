package grpc

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/api/gen/authpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthProvider interface {
	Register(ctx context.Context, login, password string) (*domain.User, error)
	Login(ctx context.Context, login, password string) (*domain.Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (*domain.Tokens, error)
}

type AuthServer struct {
	authpb.UnimplementedAuthServer
	Provider AuthProvider
}

func (a *AuthServer) Register(ctx context.Context, req *authpb.RegisterReq) (*authpb.RegisterResp, error) {
	//Validate
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password is required")
	}
	//Get result
	result, err := a.Provider.Register(ctx, req.Login, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	//Send response
	return &authpb.RegisterResp{Registred: (result.Uuid != uuid.Nil)}, nil
}

func (a *AuthServer) Login(ctx context.Context, req *authpb.LoginReq) (*authpb.LoginResp, error) {
	//Validate
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password is required")
	}
	//Get result
	tokens, err := a.Provider.Login(ctx, req.Login, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	//Send response
	return &authpb.LoginResp{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}, nil
}

func (a *AuthServer) Refresh(ctx context.Context, req *authpb.RefreshReq) (*authpb.RefreshResp, error) {
	//Validate
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}
	//Get result
	newTokens, err := a.Provider.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &authpb.RefreshResp{AccessToken: newTokens.AccessToken, RefreshToken: newTokens.RefreshToken}, nil
}
