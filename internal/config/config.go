package config

import (
	"os"
)

type Config struct {
	Port             string
	DBDriver         string
	DBPath           string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	DatabaseURL      string
	MasterAPIKey     string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "wiki.db"
	}

	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	if postgresPort == "" {
		postgresPort = "5432"
	}
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")
	databaseURL := os.Getenv("DATABASE_URL")

	masterAPIKey := os.Getenv("API_KEY")

	return &Config{
		Port:             port,
		DBDriver:         dbDriver,
		DBPath:           dbPath,
		PostgresHost:     postgresHost,
		PostgresPort:     postgresPort,
		PostgresUser:     postgresUser,
		PostgresPassword: postgresPassword,
		PostgresDB:       postgresDB,
		DatabaseURL:      databaseURL,
		MasterAPIKey:     masterAPIKey,
	}
}
