package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/alexandernizov/grpcmessanger/api/gen/authpb"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
)

type Server struct {
	log *slog.Logger

	grpcAddr string
	httpAddr string

	server    *http.Server
	isRunning bool
}

func New(log *slog.Logger, grpcAddr string, httpAddr string) *Server {
	return &Server{log: log, grpcAddr: grpcAddr, httpAddr: httpAddr}
}

func (s *Server) Start() {
	const op = "http.Start"
	log := s.log.With(slog.String("op", op))

	if s.isRunning {
		log.Error("http server is already running")
		return
	}

	conn, err := grpc.NewClient(
		s.grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("cant connect to grpc server", sl.Err(err))
	}

	gwmux := runtime.NewServeMux()
	// Register Greeter
	err = authpb.RegisterAuthHandler(context.Background(), gwmux, conn)
	// err = helloworldpb.RegisterGreeterHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Error("cant register gateway", sl.Err(err))
	}

	s.server = &http.Server{
		Addr:    s.httpAddr,
		Handler: gwmux,
	}

	s.server.ListenAndServe()

	s.isRunning = true
}

func (s *Server) Stop() {
	const op = "http.Stop"
	log := s.log.With(slog.String("op", op))

	log.Info("http is stopping")

	s.server.Shutdown(context.TODO())
	s.isRunning = false
}
