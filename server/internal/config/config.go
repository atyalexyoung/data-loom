package config

import (
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	APIKey      string
	StorageType string
	StoragePath string
	PortNumber  int
}

func Load() *Config {
	log.Info("Loading configuration")
	cfg := &Config{}

	// API KEY
	if envKey := os.Getenv("MY_SERVER_KEY"); envKey != "" {
		log.Debug("Successful setting api key for server from config")
		cfg.APIKey = envKey
	} else {
		log.Debug("Couldn't read in api key from config. Setting as default of empty string")
		cfg.APIKey = ""
	}

	// STORAGE TYPE
	if sType := os.Getenv("STORAGE_TYPE"); sType != "" {
		log.Debugf("Successfully read storage type as: %s", sType)
		cfg.StorageType = sType
	} else {
		log.Debugf("Couldn't read storage type. Setting as default of blank string for no-persistence.")
		cfg.StorageType = ""
	}

	// STORAGE PATH
	if sPath := os.Getenv("STORAGE_PATH"); sPath != "" {
		log.Debugf("Successfully read storage path from config as: %s", sPath)
		cfg.StoragePath = sPath
	} else {
		log.Debug("Couldn't read storage path from config. Setting to default of ./tmp/data")
		cfg.StoragePath = "./tmp/data"
	}

	// PORT NUMBER
	if portNumber := os.Getenv("PORT_NUMBER"); portNumber != "" {
		p, err := strconv.Atoi(portNumber)
		if err != nil || p < 1 || p > 65535 {
			log.Fatalf("Invalid PORT_NUMBER: %s. Must be 1-65535.", portNumber)
		}
		log.Debugf("Successfully read PORT_NUMBER from config as: %s", portNumber)
		cfg.PortNumber = p
	} else {
		log.Debug("PORT_NUMBER not set. Using default of 8080")
		cfg.PortNumber = 8080
	}

	return cfg
}
