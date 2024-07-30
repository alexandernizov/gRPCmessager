package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	gauth "github.com/alexandernizov/grpcmessanger/internal/grpc/auth"
	gchat "github.com/alexandernizov/grpcmessanger/internal/grpc/chat"
	"github.com/alexandernizov/grpcmessanger/internal/lib/logger/sl"
	sauth "github.com/alexandernizov/grpcmessanger/internal/services/auth"
	schat "github.com/alexandernizov/grpcmessanger/internal/services/chat"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
	"github.com/alexandernizov/grpcmessanger/internal/storage/redis"
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
	//1. Редис
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

	//2. Постгрес
	postgresAuth, err := postgres.NewPostgresAuthStorage(
		log,
		cfg.Postgres.Address,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBname,
		cfg.Postgres.MigrationsPath,
	)
	if err != nil {
		slog.Error("can't connect to postgres", sl.Err(err))
		panic(err.Error())
	}
	defer postgresAuth.Close()

	//Инициилизировать сервисный слой
	chatService := schat.NewChatService(log, redisChat, cfg.Chat.ChatTTL)
	authService := sauth.NewAuthService(log, postgresAuth, cfg.User.JwtTTL, []byte(cfg.User.JwtSecret))

	//Запустить приложение
	grpcChatServer := gchat.NewGrpcServer(log, cfg.ChatGrpc.Address, cfg.ChatGrpc.Port, cfg.ChatGrpc.RequestTimeout, chatService, cfg.User.JwtSecret)
	grpcChatServer.Start()

	grpcAuthServer := gauth.NewGrpcServer(log, cfg.AuthGrpc.Address, cfg.AuthGrpc.Port, cfg.AuthGrpc.RequestTimeout, authService)
	grpcAuthServer.Start()

	//Остановить приложение
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop

	log.Info("stopping application")
	grpcAuthServer.Stop()
	grpcChatServer.Stop()
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
