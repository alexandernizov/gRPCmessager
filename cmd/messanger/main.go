package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	"github.com/alexandernizov/grpcmessanger/internal/grpc"
	"github.com/alexandernizov/grpcmessanger/internal/services/auth"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {
	//Config
	cfg := config.MustLoad()

	//Logger
	log := setupLogger(cfg.Env)
	log.Info("starting application", slog.String("env", cfg.Env))

	//Connect postgres
	pgOpt := postgres.PostgresOptions{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBname:   cfg.Postgres.DBname,
	}
	pgDB, err := postgres.New(log, pgOpt)
	if err != nil {
		panic("can't connect to postgres")
	}

	//Auth Service
	authService := auth.NewAuthService(log, pgDB, cfg.User.JwtTTL, []byte(cfg.User.JwtSecret))

	//Start Grpc Server
	server := grpc.NewServer(log)
	gOpt := grpc.ServerOptions{
		Address:        cfg.Grpc.Address + ":" + cfg.Grpc.Port,
		RequestTimeout: cfg.Grpc.RequestTimeout,

		AuthProvider: authService,
	}
	server.Start(gOpt)

	//Stop application
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	log.Info("stopping application")
	server.Stop()
	log.Info("application stopped")

	// //Инициилизировать сторедж
	// //1. Редис
	// redisChat, err := redis.NewRedisChatStorage(
	// 	log,
	// 	cfg.Redis.Addr,
	// 	cfg.Redis.Port,
	// 	cfg.Redis.Password,
	// 	cfg.Redis.Db,
	// 	cfg.Chat.MaxChatsCount,
	// 	cfg.Chat.MaxMessagesPerChat,
	// )

	// if err != nil {
	// 	panic("cant initialize redis storage")
	// }

	// //2. Постгрес
	// postgresAuth, err := postgres.NewPostgresAuthStorage(
	// 	log,
	// 	cfg.Postgres.Address,
	// 	cfg.Postgres.Port,
	// 	cfg.Postgres.User,
	// 	cfg.Postgres.Password,
	// 	cfg.Postgres.DBname,
	// 	cfg.Postgres.MigrationsPath,
	// )
	// if err != nil {
	// 	slog.Error("can't connect to postgres", sl.Err(err))
	// 	panic(err.Error())
	// }
	// defer postgresAuth.Close()

	// //Инициилизировать сервисный слой
	// chatService := schat.NewChatService(log, redisChat, cfg.Chat.ChatTTL)
	// authService := sauth.NewAuthService(log, postgresAuth, cfg.User.JwtTTL, []byte(cfg.User.JwtSecret))

	// //Запустить приложение
	// grpcChatServer := gchat.NewGrpcServer(log, cfg.ChatGrpc.Address, cfg.ChatGrpc.Port, cfg.ChatGrpc.RequestTimeout, chatService, cfg.User.JwtSecret)
	// grpcChatServer.Start()

	// grpcAuthServer := gauth.NewGrpcServer(log, cfg.AuthGrpc.Address, cfg.AuthGrpc.Port, cfg.AuthGrpc.RequestTimeout, authService)
	// grpcAuthServer.Start()
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
