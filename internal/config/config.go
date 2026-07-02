package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                      string
	Environment               string
	DatabaseURL               string
	MaxConnections            int
	JWTSecret                 string
	AdminAPIKey               string
	OAuthStateSecret          string
	OAuthStateTTLSeconds      int
	WebSocketAllowedOrigins   []string
	CORSAllowedOrigins        []string
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	SessionStoreMode          string
	RunMigrations             bool
	AutoMigrate               bool
	MigrationsDir             string
	DBMaxOpenConns            int
	DBMaxIdleConns            int
	DBConnMaxLifetimeSeconds  int
	IdempotencyTTLSeconds     int
	OutboxPollIntervalSeconds int
	OutboxBatchSize           int
	OutboxMaxAttempts         int
	MetricsEnabled            bool
	OtelServiceName           string
	OtelExporterEndpoint      string
	OtelExporterInsecure      bool
	TokenExpiry               time.Duration
	PlatformCommission        float64
	UserBRatePerMinute        float64
	LogLevel                  string
	GoogleClientID            string
	GoogleClientSecret        string
	GoogleRedirectURL         string
	AnthropicAPIKey           string
	OpenAIAPIKey              string
	CustomLLMBaseURL          string
	CustomLLMModel            string
}

func LoadConfig() *Config {
	jwtSecret := getEnv("JWT_SECRET", "")
	maxConnections := getEnvInt("DB_MAX_CONNECTIONS", 25)
	return &Config{
		Port:                      getEnv("PORT", "8080"),
		Environment:               getEnv("ENVIRONMENT", "development"),
		DatabaseURL:               getEnv("DATABASE_URL", ""),
		MaxConnections:            maxConnections,
		JWTSecret:                 jwtSecret,
		AdminAPIKey:               getEnv("ADMIN_API_KEY", ""),
		OAuthStateSecret:          getEnv("OAUTH_STATE_SECRET", ""),
		OAuthStateTTLSeconds:      getEnvInt("OAUTH_STATE_TTL_SECONDS", 600),
		WebSocketAllowedOrigins:   getEnvCSV("WEBSOCKET_ALLOWED_ORIGINS", ""),
		CORSAllowedOrigins:        getEnvCSV("CORS_ALLOWED_ORIGINS", ""),
		RedisAddr:                 getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             getEnv("REDIS_PASSWORD", ""),
		RedisDB:                   getEnvInt("REDIS_DB", 0),
		SessionStoreMode:          getEnv("SESSION_STORE_MODE", "hybrid"),
		RunMigrations:             getEnvBool("RUN_MIGRATIONS", true),
		AutoMigrate:               getEnvBool("AUTO_MIGRATE", true),
		MigrationsDir:             getEnv("MIGRATIONS_DIR", "migrations"),
		DBMaxOpenConns:            getEnvInt("DB_MAX_OPEN_CONNS", maxConnections),
		DBMaxIdleConns:            getEnvInt("DB_MAX_IDLE_CONNS", maxConnections),
		DBConnMaxLifetimeSeconds:  getEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 300),
		IdempotencyTTLSeconds:     getEnvInt("IDEMPOTENCY_TTL_SECONDS", 86400),
		OutboxPollIntervalSeconds: getEnvInt("OUTBOX_POLL_INTERVAL_SECONDS", 5),
		OutboxBatchSize:           getEnvInt("OUTBOX_BATCH_SIZE", 10),
		OutboxMaxAttempts:         getEnvInt("OUTBOX_MAX_ATTEMPTS", 5),
		MetricsEnabled:            getEnvBool("METRICS_ENABLED", true),
		OtelServiceName:           getEnv("OTEL_SERVICE_NAME", "vk_backend"),
		OtelExporterEndpoint:      getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OtelExporterInsecure:      getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		TokenExpiry:               time.Duration(getEnvInt("TOKEN_EXPIRY", 86400)) * time.Second,
		PlatformCommission:        getEnvFloat("PLATFORM_COMMISSION_PERCENTAGE", 20.0),
		UserBRatePerMinute:        getEnvFloat("USERB_RATE_PER_MINUTE", 5.0),
		LogLevel:                  getEnv("LOG_LEVEL", "INFO"),
		GoogleClientID:            getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:        getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:         getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		AnthropicAPIKey:           getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:              getEnv("OPENAI_API_KEY", ""),
		CustomLLMBaseURL:          getEnv("CUSTOM_LLM_BASE_URL", "http://localhost:11434/v1"), // Default to Ollama
		CustomLLMModel:            getEnv("CUSTOM_LLM_MODEL", "llama3"),
	}
}

// Validate fails fast on missing or insecure required configuration instead
// of silently falling back to defaults that were historically exploitable
// (JWT_SECRET="secret", a hardcoded DATABASE_URL, OAuthStateSecret reusing
// JWTSecret).
func (c *Config) Validate() error {
	var missing []string
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.OAuthStateSecret == "" {
		missing = append(missing, "OAUTH_STATE_SECRET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}

	if c.Environment == "production" {
		weak := map[string]string{
			"JWT_SECRET":         c.JWTSecret,
			"OAUTH_STATE_SECRET": c.OAuthStateSecret,
		}
		// ADMIN_API_KEY is optional (empty disables the admin endpoints
		// entirely), but if it's set at all it must not be weak.
		if c.AdminAPIKey != "" {
			weak["ADMIN_API_KEY"] = c.AdminAPIKey
		}
		for name, val := range weak {
			if len(val) < 32 || strings.Contains(val, "change-in-production") || val == "secret" {
				return fmt.Errorf("%s is missing or too weak for production (want a random value >= 32 chars)", name)
			}
		}
		if c.OAuthStateSecret == c.JWTSecret {
			return fmt.Errorf("OAUTH_STATE_SECRET must not equal JWT_SECRET")
		}
	}
	return nil
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

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			return parsed
		}
	}
	return def
}

func getEnvCSV(key string, def string) []string {
	value := os.Getenv(key)
	if value == "" {
		value = def
	}
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
