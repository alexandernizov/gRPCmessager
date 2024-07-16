package grpc

import (
	"context"
	"errors"

	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/authpb"
	"github.com/alexandernizov/grpcmessanger/internal/domain/errs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthProvider interface {
	Register(ctx context.Context, name, password string) (bool, error)
	Login(ctx context.Context, name, password string) (string, error)
}

type authServer struct {
	authpb.UnimplementedAuthServer
	provider AuthProvider
}

func (a *authServer) Register(ctx context.Context, req *authpb.RegisterReq) (*authpb.RegisterResp, error) {
	if req.Name == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "")
	}

	res, err := a.provider.Register(ctx, req.Name, req.Password)

	if errors.Is(err, errs.ErrUserAlreadyExists) {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &authpb.RegisterResp{Registred: res}, nil
}

func (a *authServer) Login(ctx context.Context, req *authpb.LoginReq) (*authpb.LoginResp, error) {
	if req.Name == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "")
	}

	uuid, err := a.provider.Login(ctx, req.Name, req.Password)
	if errors.Is(err, errs.ErrUserNotFound) {
		return nil, status.Error(codes.InvalidArgument, "")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &authpb.LoginResp{SessionUuid: uuid}, err
}
