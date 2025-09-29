package config

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	APIKey      string
	StorageType string
	StoragePath string
}

func Load() *Config {
	log.Info("Loading configuration")
	cfg := &Config{}

	if envKey := os.Getenv("MY_SERVER_KEY"); envKey != "" {
		log.Debug("Successful setting api key for server from config")
		cfg.APIKey = envKey
	} else {
		log.Debug("Couldn't read in api key from config. Setting as default of empty string")
		cfg.APIKey = ""
	}

	if sType := os.Getenv("STORAGE_TYPE"); sType != "" {
		log.Debugf("Successfully read storage type as: %s", sType)
		cfg.StorageType = sType
	} else {
		log.Debugf("Couldn't read storage type. Setting as default of no persistence")
		cfg.StorageType = ""
	}

	if sPath := os.Getenv("STORAGE_PATH"); sPath != "" {
		log.Debugf("Successfully read storage path from config as: %s", sPath)
		cfg.StoragePath = sPath
	} else {
		log.Debug("Couldn't read storage path from config. Setting to default of ./tmp/data")
		cfg.StoragePath = "./tmp/data"
	}

	return cfg
}
