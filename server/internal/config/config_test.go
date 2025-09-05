package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("MY_SERVER_KEY", "")
	t.Setenv("STORAGE_TYPE", "")
	t.Setenv("STORAGE_PATH", "")

	cfg := Load()

	assert.Equal(t, "data-loom-api-key", cfg.APIKey)
	assert.Equal(t, "badger", cfg.StorageType)
	assert.Equal(t, "./tmp/data", cfg.StoragePath)
}

func TestLoad_WithEnvVars(t *testing.T) {
	t.Setenv("MY_SERVER_KEY", "test-key")
	t.Setenv("STORAGE_TYPE", "sqlite")
	t.Setenv("STORAGE_PATH", "/var/data")

	cfg := Load()

	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, "sqlite", cfg.StorageType)
	assert.Equal(t, "/var/data", cfg.StoragePath)
}
