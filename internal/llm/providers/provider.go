package providers

import (
	"context"
	"time"
)

type Message struct {
	Role    string
	Content string
}

type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	UserId      string
}

type CompletionResponse struct {
	Content    string
	TokensUsed int
	Model      string
	Provider   string
	Latency    time.Duration
	Cached     bool
}

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Name() string
	HealthCheck(ctx context.Context) error
}
