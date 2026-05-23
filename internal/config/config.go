package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	DGISAPIKey    string
	DatabasePath  string
	OutputDir     string
	MaxPages      int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		if os.Getenv("TELEGRAM_TOKEN") == "" {
			log.Println(".env file not found, using environment variables")
		}
	}

	cfg := &Config{
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),
		DGISAPIKey:    getEnv("DGIS_API_KEY", ""),
		DatabasePath:  getEnv("DATABASE_PATH", "./data/parser.db"),
		OutputDir:     getEnv("OUTPUT_DIR", "./exports"),
		MaxPages:      getEnvInt("DGIS_MAX_PAGES", 5),
	}

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN is required. Set it in the .env file")
	}
	if cfg.DGISAPIKey == "" {
		log.Fatal("DGIS_API_KEY is required. Get a 2GIS API key and set it in the .env file")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
