package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/alexandernizov/grpcmessanger/internal/grpc/chat/chatpb"
	"github.com/alexandernizov/grpcmessanger/internal/lib/logger/sl"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

//go:generate protoc -I ../../api/protos ../../api/protos/chat_service.proto --go_out=. --go-grpc_out=.

type GrpcServer struct {
	log *slog.Logger

	grpcServer     *grpc.Server
	address        string
	port           int
	requestTimeout time.Duration
	isRunning      bool

	chatService ChatProvider
	secret      string
}

func NewGrpcServer(
	log *slog.Logger,
	address string,
	port int,
	requestTimeout time.Duration,
	chatProvider ChatProvider,
	secret string,
) *GrpcServer {
	return &GrpcServer{
		log:            log,
		address:        address,
		port:           port,
		requestTimeout: requestTimeout,
		chatService:    chatProvider,
		secret:         secret,
	}
}

func (s *GrpcServer) Start() {
	const op = "grpc.Start"

	log := s.log.With(slog.String("op", op))

	if s.isRunning {
		log.Error("grpc server is already started")
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%v", s.address, s.port))
	if err != nil {
		log.Error("can't make the listener", sl.Err(err))
	}

	s.grpcServer = grpc.NewServer(grpc.ChainUnaryInterceptor(unaryLoggingInterceptor(s.log)))

	chatServer := &ChatServer{Provider: s.chatService, Secret: s.secret}

	chatpb.RegisterChatServer(s.grpcServer, chatServer)

	reflection.Register(s.grpcServer)

	log.Info("grpc server is running")

	s.isRunning = true

	go func() {
		s.grpcServer.Serve(listener)
	}()
}

func (s *GrpcServer) Stop() {
	const op = "grpc.Stop"
	log := s.log.With(slog.String("op", op))

	log.Info("grpc is stopping")

	s.grpcServer.GracefulStop()
	s.isRunning = false
}

func unaryLoggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		log.Info(fmt.Sprintf("Request: %s", info.FullMethod))

		resp, err := handler(ctx, req)

		if err != nil {
			st := status.Convert(err)
			reqJSON, _ := json.Marshal(req)

			switch st.Code() {
			case codes.Unauthenticated:
				log.Warn(fmt.Sprintf("Unauthenticated try: %s Request: %s", info.FullMethod, reqJSON))
			default:
				log.Warn(fmt.Sprintf("Request error: %s, %s", st.Code().String(), reqJSON))
			}
		}

		return resp, err
	}
}
