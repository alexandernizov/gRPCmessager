package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	gchat "github.com/alexandernizov/grpcmessanger/internal/grpc/chat"
	schat "github.com/alexandernizov/grpcmessanger/internal/services/chat"
	chatstorage "github.com/alexandernizov/grpcmessanger/storage/chatStorage"
	"github.com/alexandernizov/grpcmessanger/storage/chatStorage/redis"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	//Инициализируем конфиг
	cfg := config.MustLoad()

	//Инициилазируем логгер
	log := setupLogger(cfg.Env)
	log.Info("starting application", slog.String("env", cfg.Env))

	log.Info("server params",
		slog.String("enviroment", cfg.Env),
		slog.Int("port", cfg.ChatGrpc.Port),
		slog.Duration("jwt ttl", cfg.User.JwtTTL),
	)

	//Инициилизировать сторедж
	redisChat, err := redis.NewRedisChatStorage(
		log,
		cfg.Redis.Addr,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.Db,
		cfg.Chat.MaxChatsCount,
		cfg.Chat.MaxMessagesPerChat,
	)

	if err != nil {
		panic("cant initialize redis storage")
	}

	storage := chatstorage.NewStorage(redisChat)

	//Инициилизировать сервисный слой
	//service := sauth.NewAuthService(log, storage, cfg.User.JwtTTL, []byte(cfg.User.JwtSecret))
	service := schat.NewChatService(log, storage, cfg.Chat.ChatTTL)

	//Запустить приложение
	grpcServer := gchat.NewGrpcServer(log, cfg.ChatGrpc.Address, cfg.ChatGrpc.Port, cfg.ChatGrpc.RequestTimeout, service, cfg.User.JwtSecret)
	grpcServer.Start()

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
