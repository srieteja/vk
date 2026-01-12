package services

import (
	"context"
	"encoding/json"
	"vk_backend/internal/config"
	"vk_backend/internal/llm"
	"vk_backend/internal/llm/cache"
	"vk_backend/internal/llm/circuit"
	"vk_backend/internal/llm/providers"
	"vk_backend/internal/llm/ratelimit"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

var llmClient *llm.Client

type LLMService struct {
	db *gorm.DB
}

func NewLLMService(db *gorm.DB) *LLMService {
	return &LLMService{
		db: db,
	}
}

func (s *LLMService) Start(cfg *config.Config) {
	// Initialize cache
	redisCache, err := cache.NewRedisCache("localhost:6379")
	if err != nil {
		log.Printf("Warning: Redis cache unavailable: %v", err)
	}

	// Initialize providers
	anthropicProvider := providers.NewAnthropicProvider(
		cfg.AnthropicAPIKey,
		"claude-haiku-4-5-20251001",
		1024,
	)

	// Custom Provider (e.g. Ollama/vLLM)
	// We use "ollama" as the API key if not provided, as local servers often don't require one
	customProvider := providers.NewCustomProvider(
		"ollama",
		cfg.CustomLLMBaseURL,
		cfg.CustomLLMModel,
	)

	// Initialize rate limiter
	rateLimiter := ratelimit.NewRateLimiter(
		1000, // global RPM
		50,   // global burst
		10,   // per-user RPM
		3,    // per-user burst
	)
	rateLimiter.StartCleanup(5 * time.Minute)

	// Initialize circuit breaker
	circuitBreaker := circuit.NewCircuitBreaker(
		5,              // max failures
		30*time.Second, // timeout
		30*time.Second, // reset timeout
	)

	// Initialize LLM client
	llmClient = llm.NewClient(llm.Config{
		Primary:        customProvider,    // Try your custom model first
		Fallback:       anthropicProvider, // Fallback to Claude
		Cache:          redisCache,
		RateLimiter:    rateLimiter,
		PrimaryBreaker: circuitBreaker,
		CacheTTL:       3600,
		CacheEnabled:   true,
	})

	log.Println("LLM Service initialized")
}

type ChatRequest struct {
	UserID  string              `json:"user_id"`
	Message string              `json:"message"`
	History []providers.Message `json:"history,omitempty"`
}

type ChatResponse struct {
	Response   string  `json:"response"`
	TokensUsed int     `json:"tokens_used"`
	Provider   string  `json:"provider"`
	Latency    float64 `json:"latency_ms"`
	Cached     bool    `json:"cached"`
}

// HandleChat handles chat requests to the LLM service
func HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Build messages
	messages := append(req.History, providers.Message{
		Role:    "user",
		Content: req.Message,
	})

	// Call LLM
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Check if client is initialized
	if llmClient == nil {
		http.Error(w, "LLM service not initialized", http.StatusServiceUnavailable)
		return
	}

	response, err := llmClient.Complete(ctx, providers.CompletionRequest{
		Messages: messages,
		UserId:   req.UserID,
	})

	if err != nil {
		log.Printf("Error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(ChatResponse{
		Response:   response.Content,
		TokensUsed: response.TokensUsed,
		Provider:   response.Provider,
		Latency:    float64(response.Latency.Milliseconds()),
		Cached:     response.Cached,
	})
	if err != nil {
		return
	}
}

// HandleHealth handles health check requests for the LLM service
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if llmClient == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if err := llmClient.HealthCheck(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		err := json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		if err != nil {
			return
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
	if err != nil {
		return
	}
}