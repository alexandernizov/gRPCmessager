package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	"github.com/alexandernizov/grpcmessanger/internal/grpc"
	"github.com/alexandernizov/grpcmessanger/internal/services/auth"
	"github.com/alexandernizov/grpcmessanger/internal/services/chat"
	"github.com/alexandernizov/grpcmessanger/storage"
	"github.com/alexandernizov/grpcmessanger/storage/inmemory"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	//Иницилизировать конфиг
	cfg := config.MustLoad()

	//Инициализировать логгер
	log := setupLogger(cfg.Env)
	log.Info("starting application", slog.String("env", cfg.Env))

	log.Info("server params",
		slog.Int("port", cfg.Port),
		slog.Int("maximum chats", cfg.MaxChatsCount),
		slog.Int("messages per chat", cfg.MaxMessagesPerChat),
	)

	//Инициализировать сторейдж
	inmemAuthStorage := inmemory.NewAuthMemStorage(cfg.SessionTTL)
	inmemChatStorage := inmemory.NewChatMemStorage(cfg.MaxChatsCount, cfg.MaxMessagesPerChat)
	storage := storage.NewStorage(inmemAuthStorage, inmemChatStorage)

	//Инициилизировать service слой
	authService := auth.NewAuthService(storage)
	chatService := chat.NewChatService(storage, int(cfg.ChatTTL))

	//Запустить приложение
	grpcServer := grpc.NewGrpcServer(log, cfg.GrpcConfig.Port, authService, authService, chatService)
	go grpcServer.Start()
	//Остановить приложение
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	log.Info("stopping application")
	grpcServer.Stop()
	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		panic("unknown enviroment")
	}

	return log
}
