package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	Environment        string
	DatabaseURL        string
	MaxConnections     int
	JWTSecret          string
	TokenExpiry        time.Duration
	PlatformCommission float64
	UserBRatePerMinute float64
	LogLevel           string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	AnthropicAPIKey    string
	OpenAIAPIKey       string
	CustomLLMBaseURL   string
	CustomLLMModel     string
}

func LoadConfig() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Environment:        getEnv("ENVIRONMENT", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://srie:qwerty@localhost:5432/vk_db"),
		MaxConnections:     getEnvInt("DB_MAX_CONNECTIONS", 25),
		JWTSecret:          getEnv("JWT_SECRET", "secret"),
		TokenExpiry:        time.Duration(getEnvInt("TOKEN_EXPIRY", 86400)) * time.Second,
		PlatformCommission: getEnvFloat("PLATFORM_COMMISSION_PERCENTAGE", 20.0),
		UserBRatePerMinute: getEnvFloat("USERB_RATE_PER_MINUTE", 5.0),
		LogLevel:           getEnv("LOG_LEVEL", "INFO"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		AnthropicAPIKey:    getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:       getEnv("OPENAI_API_KEY", ""),
		CustomLLMBaseURL:   getEnv("CUSTOM_LLM_BASE_URL", "http://localhost:11434/v1"), // Default to Ollama
		CustomLLMModel:     getEnv("CUSTOM_LLM_MODEL", "llama3"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
