package auth

import (
	"context"

	"github.com/alexandernizov/grpcmessanger/internal/grpc/auth/authpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthProvider interface {
	Register(ctx context.Context, login, password string) (bool, error)
	Login(ctx context.Context, login, password string) (string, error)
	Validate(ctx context.Context, token string) (bool, error)
}

type AuthServer struct {
	authpb.UnimplementedAuthServer
	Provider AuthProvider
}

func (a *AuthServer) Register(ctx context.Context, req *authpb.RegisterReq) (*authpb.RegisterResp, error) {
	//Validate request
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password is required")
	}
	//Get result
	result, err := a.Provider.Register(ctx, req.Login, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	//Send response
	return &authpb.RegisterResp{Registred: result}, nil
}

func (a *AuthServer) Login(ctx context.Context, req *authpb.LoginReq) (*authpb.LoginResp, error) {
	//Validate request
	if req.Login == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "login and password is required")
	}
	//Get result
	token, err := a.Provider.Login(ctx, req.Login, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	//Send response
	return &authpb.LoginResp{Token: token}, nil
}
func (a *AuthServer) Validate(ctx context.Context, req *authpb.ValidateReq) (*authpb.ValidateResp, error) {
	//Validate request
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "jwt token is required")
	}
	//Get result
	result, err := a.Provider.Validate(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	//Send response
	return &authpb.ValidateResp{Valid: result}, nil
}
