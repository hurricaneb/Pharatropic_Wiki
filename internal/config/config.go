package config

import (
	"os"
)

type Config struct {
	Port   string
	DBPath string
	MasterAPIKey string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "wiki.db"
	}

	masterAPIKey := os.Getenv("API_KEY")
	if masterAPIKey == "" {
		masterAPIKey = "wiki-secret-api-key"
	}

	return &Config{
		Port:         port,
		DBPath:       dbPath,
		MasterAPIKey: masterAPIKey,
	}
}
