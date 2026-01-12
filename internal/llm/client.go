package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"vk_backend/internal/llm/cache"
	"vk_backend/internal/llm/circuit"
	"vk_backend/internal/llm/providers"
	"vk_backend/internal/llm/ratelimit"
)

type Client struct {
	primary        providers.Provider
	fallback       providers.Provider
	cache          cache.Cache
	rateLimiter    *ratelimit.RateLimiter
	primaryBreaker *circuit.CircuitBreaker
	cacheTTL       int
	cacheEnabled   bool
}

type Config struct {
	Primary        providers.Provider
	Fallback       providers.Provider
	Cache          cache.Cache
	RateLimiter    *ratelimit.RateLimiter
	PrimaryBreaker *circuit.CircuitBreaker
	CacheTTL       int
	CacheEnabled   bool
}

func NewClient(cfg Config) *Client {
	return &Client{
		primary:        cfg.Primary,
		fallback:       cfg.Fallback,
		cache:          cfg.Cache,
		rateLimiter:    cfg.RateLimiter,
		primaryBreaker: cfg.PrimaryBreaker,
		cacheTTL:       cfg.CacheTTL,
		cacheEnabled:   cfg.CacheEnabled,
	}
}

func (c *Client) Complete(ctx context.Context, req providers.CompletionRequest) (*providers.CompletionResponse, error) {
	// Rate limiting
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Allow(ctx, req.UserId); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Check cache
	if c.cacheEnabled && c.cache != nil {
		cacheKey := cache.GenerateCacheKey(c.messagesToInterface(req.Messages), c.primary.Name())
		if cachedResponse, err := c.cache.Get(ctx, cacheKey); err == nil {
			var response providers.CompletionResponse
			if err := json.Unmarshal([]byte(cachedResponse), &response); err == nil {
				response.Cached = true
				log.Printf("Cache hit for user %s", req.UserId)
				return &response, nil
			}
		}
	}

	// Try primary provider with circuit breaker
	var response *providers.CompletionResponse
	var err error

	if c.primaryBreaker != nil {
		err = c.primaryBreaker.Execute(ctx, func() error {
			response, err = c.primary.Complete(ctx, req)
			return err
		})
	} else {
		response, err = c.primary.Complete(ctx, req)
	}

	// Fallback to secondary provider if primary fails
	if err != nil {
		log.Printf("Primary provider failed: %v, trying fallback", err)
		if c.fallback != nil {
			response, err = c.fallback.Complete(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("both providers failed: %w", err)
			}
		} else {
			return nil, fmt.Errorf("primary provider failed and no fallback configured: %w", err)
		}
	}

	// Cache successful response
	if c.cacheEnabled && c.cache != nil && response != nil {
		cacheKey := cache.GenerateCacheKey(c.messagesToInterface(req.Messages), response.Provider)
		if responseJSON, err := json.Marshal(response); err == nil {
			_ = c.cache.Set(ctx, cacheKey, string(responseJSON), c.cacheTTL)
		}
	}

	return response, nil
}

func (c *Client) messagesToInterface(messages []providers.Message) []interface{} {
	result := make([]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	return result
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.primary.HealthCheck(ctx); err != nil {
		return fmt.Errorf("primary provider unhealthy: %w", err)
	}

	if c.fallback != nil {
		if err := c.fallback.HealthCheck(ctx); err != nil {
			log.Printf("Warning: fallback provider unhealthy: %v", err)
		}
	}

	return nil
}
