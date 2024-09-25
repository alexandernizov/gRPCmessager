package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"reflect"
	"time"

	"github.com/alexandernizov/grpcmessanger/api/gen/authpb"
	"github.com/alexandernizov/grpcmessanger/api/gen/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/jwt"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

////go:generate protoc -I ../../api/protos ../../api/protos/auth_service.proto --go_out=../../api/ --go-grpc_out=../../api/ --grpc-gateway_out=../../api/
////go:generate protoc -I ../../api/protos ../../api/protos/chat_service.proto --go_out=../../api/ --go-grpc_out=../../api/ --grpc-gateway_out=../../api/

var (
	ErrServerIsAlreadyRunning = errors.New("server is already running")
)

type Server struct {
	log       *slog.Logger
	server    *grpc.Server
	isRunning bool
}

func NewServer(log *slog.Logger) *Server {
	return &Server{log: log}
}

type ServerOptions struct {
	Address        string
	RequestTimeout time.Duration
	JwtSecret      []byte

	AuthProvider
	ChatProvider
}

func (s *Server) Start(opt ServerOptions) {
	const op = "grpc.Start"
	log := s.log.With(slog.String("op", op))

	if s.isRunning {
		log.Error("can't start server", sl.Err(ErrServerIsAlreadyRunning))
		return
	}

	listener, err := net.Listen("tcp", opt.Address)
	if err != nil {
		log.Error("can't make listener", sl.Err(err))
	}

	s.server = grpc.NewServer(grpc.ChainUnaryInterceptor(
		unaryLoggingInterceptor(s.log),
		unaryAuthInterceptor(s.log, opt.JwtSecret),
	))
	authpb.RegisterAuthServer(s.server, &AuthServer{Provider: opt.AuthProvider})
	chatpb.RegisterChatServer(s.server, &ChatServer{Provider: opt.ChatProvider, Secret: string(opt.JwtSecret)})
	reflection.Register(s.server)

	log.Info("grpc server is running")

	s.isRunning = true

	go func() {
		err := s.server.Serve(listener)
		if err != nil {
			s.log.Error("error with grpc serve listener", sl.Err(err))
		}
	}()
}

func (s *Server) Stop() {
	const op = "grpc.Stop"
	log := s.log.With(slog.String("op", op))

	log.Info("grpc is stopping")

	s.server.GracefulStop()
	s.isRunning = false
}

func unaryLoggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		log.Info(fmt.Sprintf("Request: %s", info.FullMethod))

		resp, err := handler(ctx, req)

		if err != nil {
			st := status.Convert(err)
			reqJSON, _ := json.Marshal(req)
			log.Warn(fmt.Sprintf("Request error: %s, %s", st.Code().String(), reqJSON))
		}

		return resp, err
	}
}

func unaryAuthInterceptor(log *slog.Logger, jwtSecret []byte) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		skip := make(map[string]bool)
		skip["/authpb.Auth/Register"] = true
		skip["/authpb.Auth/Login"] = true
		skip["/authpb.Auth/Refresh"] = true

		if _, ok := skip[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		// Используем рефлексию для получения поля "token"
		v := reflect.ValueOf(req)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		// Проверяем, есть ли поле "token" в сообщении
		tokenField := v.FieldByName("Token")
		if !tokenField.IsValid() {
			return nil, status.Errorf(codes.Unauthenticated, "token field is missing in request")
		}

		token, ok := tokenField.Interface().(string)
		if !ok || token == "" {
			return nil, status.Errorf(codes.Unauthenticated, "token is invalid or missing")
		}

		ok, err := jwt.ValidateToken(token, jwtSecret)
		if !ok || err != nil {
			log.Warn("someone trying to get access with invalid token", slog.String("token", token))
			return nil, status.Errorf(codes.Unauthenticated, "token is invalid")
		}

		return handler(ctx, req)
	}
}
