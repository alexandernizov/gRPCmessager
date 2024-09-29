package config_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/alexandernizov/grpcmessanger/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMustLoad(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		input      string
	}{
		{
			name:       "success",
			configPath: "test_config.yaml",
			input:      `env: "test"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := createTempConfigFile(t, tt.configPath, tt.input)
			defer os.Remove(tempFile)
			os.Args = []string{"test", "-config", tt.configPath}

			cfg := config.MustLoad()

			if len(tt.input) > 0 {
				assert.NotNil(t, cfg, "config is empty")
			} else {
				assert.Nil(t, cfg, "config should be empty")
			}
		})
	}
}

func createTempConfigFile(t *testing.T, path string, content string) string {
	tmpFile, err := os.Create(path)
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
