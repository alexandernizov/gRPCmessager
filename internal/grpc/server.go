package grpc

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/authpb"
	"github.com/alexandernizov/grpcmessanger/api/gen/messanger/chatpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type GrpcServer struct {
	log        *slog.Logger
	grpcServer *grpc.Server
	port       int

	authValidator AuthValidator
	authService   AuthProvider
	chatService   ChatProvider
}

func NewGrpcServer(
	log *slog.Logger,
	port int,
	authValidator AuthValidator,
	authProvider AuthProvider,
	chatProvider ChatProvider,
) *GrpcServer {
	return &GrpcServer{
		log:           log,
		port:          port,
		authValidator: authValidator,
		authService:   authProvider,
		chatService:   chatProvider,
	}
}

func (s *GrpcServer) Start() {
	const op = "grpc.Start"
	log := s.log.With(slog.String("op", op))

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		log.Error("can't make the listener", fmt.Sprintf("localhost:%d", s.port), err.Error())
		panic(err.Error())
	}

	authInterceptor := NewAuthInterceptor(s.authValidator)
	loggingInterceptor := unaryLoggingInterceptor(s.log)

	s.grpcServer = grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{MaxConnectionIdle: 5 * time.Minute}),
		grpc.ChainUnaryInterceptor(loggingInterceptor, authInterceptor.Unary()),
	)

	authServer := &authServer{provider: s.authService}
	chatServer := &chatServer{provider: s.chatService}

	authpb.RegisterAuthServer(s.grpcServer, authServer)
	chatpb.RegisterChatServer(s.grpcServer, chatServer)

	reflection.Register(s.grpcServer)

	log.Info("grpc server is running", slog.String("addr", listener.Addr().String()))

	if err := s.grpcServer.Serve(listener); err != nil {
		log.Error("%s: %w", op, err)
		return
	}
}

func (s *GrpcServer) Stop() {
	const op = "grpc.Stop"
	log := s.log.With(slog.String("op", op))
	log.Info("grpc is stopping")
	s.grpcServer.GracefulStop()
	log.Info("grpc stopped")
}
