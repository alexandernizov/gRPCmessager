package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	gauth "github.com/alexandernizov/grpcmessanger/internal/grpc/auth"
	"github.com/alexandernizov/grpcmessanger/internal/lib/logger/sl"
	sauth "github.com/alexandernizov/grpcmessanger/internal/services/auth"
	authstorage "github.com/alexandernizov/grpcmessanger/storage/authStorage"
	"github.com/alexandernizov/grpcmessanger/storage/authStorage/postgres"
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
		slog.Int("port", cfg.AuthGrpc.Port),
		slog.Duration("jwt ttl", cfg.User.JwtTTL),
	)

	//Инициилизировать сторедж
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

	storage := authstorage.NewStorage(postgresAuth)

	//Инициилизировать сервисный слой
	service := sauth.NewAuthService(log, storage, cfg.User.JwtTTL, []byte(cfg.User.JwtSecret))

	//Запустить приложение
	grpcServer := gauth.NewGrpcServer(log, cfg.AuthGrpc.Address, cfg.AuthGrpc.Port, cfg.AuthGrpc.RequestTimeout, service)
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
