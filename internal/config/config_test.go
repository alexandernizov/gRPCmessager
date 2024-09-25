package config_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	"github.com/stretchr/testify/assert"
)

const (
	testConfigPath = "test_config.yaml"
)

func createTempConfigFile(t *testing.T, content string) string {
	tmpFile, err := os.Create(testConfigPath)
	if err != nil {
		t.Fatalf("Can't create temp test config file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Can't write into temp test config file: %v", err)
	}

	fmt.Println(tmpFile.Name())

	return tmpFile.Name()
}

func TestMustLoad(t *testing.T) {
	configContent := `
env: "test"

grpc:
  address: "0.0.0.0"

http:
  address: "0.0.0.0"

chat:
  maximum_chats_count: 1

user:
  jwt_secret: 1

kafka:
  host: "0.0.0.0"

storage:
  inmemory: 1

postgres:
  host: "0.0.0.0"

redis:
  address: "0.0.0.0"
`
	tempFile := createTempConfigFile(t, configContent)
	defer os.Remove(tempFile)

	os.Args = []string{"test", "-config", testConfigPath}
	cfg := config.MustLoad()

	assert.NotNil(t, cfg)
	assert.Equal(t, "test", cfg.Env)
	assert.Equal(t, "0.0.0.0", cfg.Grpc.Address)
	assert.Equal(t, "0.0.0.0", cfg.Http.Addr)
	assert.Equal(t, 1, cfg.Chat.MaxChatsCount)
	assert.Equal(t, "1", cfg.User.JwtSecret)
	assert.Equal(t, "0.0.0.0", cfg.Kafka.Host)
	assert.Equal(t, 1, cfg.Storage.Inmemory)
	assert.Equal(t, "0.0.0.0", cfg.Postgres.Host)
	assert.Equal(t, "0.0.0.0", cfg.Redis.Addr)
}
