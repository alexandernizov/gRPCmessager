package grpc

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/alexandernizov/grpcmessanger/internal/domain"
	"github.com/alexandernizov/grpcmessanger/internal/domain/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	AuthValidator
}

func NewAuthInterceptor(authValidator AuthValidator) *AuthInterceptor {
	return &AuthInterceptor{authValidator}
}

type AuthValidator interface {
	IsValid(ctx context.Context, name, password string) (bool, error)
	GetUser(ctx context.Context, name, password string) (*domain.User, error)
}

func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		if info.FullMethod == "/authpb.Auth/Register" {
			return handler(ctx, req)
		}
		if info.FullMethod == "/authpb.Auth/Login" {
			return handler(ctx, req)
		}

		name, pass, err := interceptor.getNamePass(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		valid, err := interceptor.IsValid(ctx, name, pass)
		if errors.Is(err, errs.ErrUserNotFound) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if !valid {
			return nil, status.Error(codes.PermissionDenied, "session is expired")
		}

		user, err := interceptor.GetUser(ctx, name, pass)
		if errors.Is(err, errs.ErrUserNotFound) {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		//user := interceptor.Get
		ctx = context.WithValue(ctx, domain.UserCtxKey{}, user)

		return handler(ctx, req)
	}
}

func (interceptor *AuthInterceptor) getNamePass(ctx context.Context) (string, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "metadata not found")
	}

	authHeaders, ok := md["authorization"]
	if !ok || len(authHeaders) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "authorization header not found")
	}

	authHeader := authHeaders[0]
	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", "", status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	encodedCreds := strings.TrimPrefix(authHeader, prefix)
	creds, err := base64.StdEncoding.DecodeString(encodedCreds)
	if err != nil {
		return "", "", status.Error(codes.Unauthenticated, "invalid base64 encoding")
	}

	credsParts := strings.SplitN(string(creds), ":", 2)
	if len(credsParts) != 2 {
		return "", "", status.Error(codes.Unauthenticated, "invalid credentials format")
	}

	username := credsParts[0]
	password := credsParts[1]

	return username, password, nil
}
