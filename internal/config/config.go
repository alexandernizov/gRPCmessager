package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env        string `yaml:"env"`
	GrpcConfig `yaml:"grpc"`
	ChatConfig `yaml:"chats"`
	UserConfig `yaml:"user"`
}

type GrpcConfig struct {
	Port           int           `yaml:"port"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
}

type ChatConfig struct {
	MaxChatsCount      int           `yaml:"maximum_chats_count"`
	MaxMessagesPerChat int           `yaml:"messages_per_chat"`
	ChatTTL            time.Duration `yaml:"chat_ttl"`
}

type UserConfig struct {
	SessionTTL time.Duration `yaml:"session_ttl"`
}

func MustLoad() *Config {
	path, port, chatCount, messagesPerChat := fetchFlags()

	if path == "" {
		path = "configs/local.yaml"
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	if port != 0 {
		cfg.Port = port
	}

	if chatCount != 0 {
		cfg.MaxChatsCount = chatCount
	}

	if messagesPerChat != 0 {
		cfg.MaxMessagesPerChat = messagesPerChat
	}

	return &cfg
}

func MustLoadByPath(path string) *Config {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("config file does not exist: " + path)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &cfg
}

func fetchFlags() (string, int, int, int) {
	var path string
	var port int
	var chatCount int
	var messagesPerChat int

	flag.StringVar(&path, "config", "", "path to config file")
	flag.IntVar(&port, "port", 0, "grpc server port")
	flag.IntVar(&chatCount, "cc", 0, "maximum chats in one moment. Must be greater than zero")
	flag.IntVar(&messagesPerChat, "mm", 0, "maximum messages per chat. Must be greater than zero")
	flag.Parse()

	return path, port, chatCount, messagesPerChat
}
