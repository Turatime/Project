package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Port        string
	Timezone    string
	DBPath      string
	LLMEndpoint string
	LLMAPIKey   string
	LLMModel    string
	EnableLIFF  bool
}

func Load() AppConfig {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("[cfg] No .env file found or error loading: %v", err)
	}

	get := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	cfg := AppConfig{
		Port:        get("PORT", "8080"),
		Timezone:    get("TZ", "Asia/Bangkok"),
		DBPath:      get("DB_PATH", "aoi.db"),
		LLMEndpoint: get("LLM_ENDPOINT", ""),
		LLMAPIKey:   get("LLM_API_KEY", ""),
		LLMModel:    get("LLM_MODEL", "gpt-4o-mini"),
		EnableLIFF:  get("ENABLE_LIFF", "false") == "true",
	}
	log.Printf("[cfg] %+v", cfg)
	return cfg
}
