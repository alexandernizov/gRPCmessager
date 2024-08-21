package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	"github.com/alexandernizov/grpcmessanger/internal/grpc"
	"github.com/alexandernizov/grpcmessanger/internal/http"
	"github.com/alexandernizov/grpcmessanger/internal/services/auth"
	"github.com/alexandernizov/grpcmessanger/internal/services/chat"
	"github.com/alexandernizov/grpcmessanger/internal/storage/postgres"
	"github.com/alexandernizov/grpcmessanger/internal/storage/redis"
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
	pgOpt := postgres.ConnectOptions{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBname:   cfg.Postgres.DBname,
	}
	pgDB, err := postgres.NewWithOptions(log, pgOpt)
	if err != nil {
		panic("can't connect to postgres")
	}

	//Auth Service
	jwt := auth.JwtParams{AccessTtl: cfg.User.JwtAccessTTL, RefreshTtl: cfg.User.JwtRefreshTTL, Secret: []byte(cfg.User.JwtSecret)}
	authService := auth.NewService(log, pgDB, jwt)

	//Connect redis
	redisOpt := redis.ConnectOptions{
		Addr:     cfg.Redis.Addr + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Db,
	}
	redisDB, err := redis.New(log, redisOpt)
	if err != nil {
		panic("can't connect to redis")
	}

	//Chat Service
	chatOpt := chat.ChatOptions{
		DefaultTtl:      cfg.Chat.ChatTTL,
		MaximumCount:    cfg.Chat.MaxChatsCount,
		MaximumMessages: cfg.Chat.MaxMessagesPerChat,
	}
	chatService := chat.NewService(log, chatOpt, redisDB)

	//Start Grpc Server
	server := grpc.NewServer(log)
	gOpt := grpc.ServerOptions{
		Address:        cfg.Grpc.Address + ":" + cfg.Grpc.Port,
		RequestTimeout: cfg.Grpc.RequestTimeout,
		JwtSecret:      []byte(cfg.User.JwtSecret),

		AuthProvider: authService,
		ChatProvider: chatService,
	}
	server.Start(gOpt)

	httpServer := http.New(log, cfg.Grpc.Address+":"+cfg.Grpc.Port, cfg.Http.Addr+":"+cfg.Http.Port)
	httpServer.Start()

	//Stop application
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	<-stop
	log.Info("stopping application")
	//	httpServer.Stop()
	server.Stop()
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
