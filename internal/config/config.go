package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string         `yaml:"env"`
	Postgres PostgresConfig `yaml:"postgres"`
	Redis    RedisConfig    `yaml:"redis"`
	User     UserConfig     `yaml:"user"`
	AuthGrpc AuthGrpcConfig `yaml:"grpc_auth"`
	ChatGrpc ChatGrpcConfig `yaml:"grpc_chat"`
	Chat     ChatConfig     `yaml:"chat"`
}

type RedisConfig struct {
	Addr     string `yaml:"address"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
}

type ChatConfig struct {
	MaxChatsCount      int           `yaml:"maximum_chats_count"`
	MaxMessagesPerChat int           `yaml:"messages_per_chat"`
	ChatTTL            time.Duration `yaml:"chat_ttl"`
}

type PostgresConfig struct {
	Address        string `yaml:"address"`
	Port           string `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	DBname         string `yaml:"dbname"`
	MigrationsPath string `yaml:"migrations_path"`
}

type AuthGrpcConfig struct {
	Address        string        `yaml:"address"`
	Port           int           `yaml:"port"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type ChatGrpcConfig struct {
	Address        string        `yaml:"address"`
	Port           int           `yaml:"port"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type UserConfig struct {
	JwtTTL    time.Duration `yaml:"jwt_ttl"`
	JwtSecret string        `yaml:"jwt_secret"`
}

func MustLoad() *Config {
	path := fetchFlags()
	if path == "" {
		path = "configs/local.yaml"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exists: " + path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}
	return &cfg
}

func fetchFlags() string {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	return configPath
}
