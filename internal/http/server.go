package http

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/alexandernizov/grpcmessanger/api/gen/authpb"
	"github.com/alexandernizov/grpcmessanger/internal/pkg/logger/sl"
)

type Server struct {
	log *slog.Logger

	grpcAddr   string
	httpAddr   string
	prometheus bool

	server    *http.Server
	isRunning bool
}

func New(options ...func(*Server)) *Server {
	server := &Server{}
	for _, option := range options {
		option(server)
	}
	return server
}

func WithLogger(log *slog.Logger) func(*Server) {
	return func(s *Server) {
		s.log = log
	}
}

func WithHttpAddr(httpAddr string) func(*Server) {
	return func(s *Server) {
		s.httpAddr = httpAddr
	}
}

func WithGrpcGateway(grpcAddr string) func(*Server) {
	return func(s *Server) {
		s.grpcAddr = grpcAddr
	}
}

func WithPrometheus() func(*Server) {
	return func(s *Server) {
		s.prometheus = true
	}
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

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", gwmux)

	s.server = &http.Server{
		Addr:    s.httpAddr,
		Handler: mux,
	}

	err = s.server.ListenAndServe()
	if err != nil {
		log.Error("error during start http server", sl.Err(err))
	}

	s.isRunning = true
}

func (s *Server) Stop() {
	const op = "http.Stop"
	log := s.log.With(slog.String("op", op))

	log.Info("http is stopping")

	err := s.server.Shutdown(context.TODO())
	if err != nil {
		log.Error("error during shutdown http server", sl.Err(err))
	}
	s.isRunning = false
}
