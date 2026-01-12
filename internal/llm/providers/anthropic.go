package providers

import (
	"context"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type AnthropicProvider struct {
	client    anthropic.Client
	model     string
	maxTokens int
}

func NewAnthropicProvider(apiKey, model string, maxTokens int) *AnthropicProvider {
	requestConfig := option.WithAPIKey(apiKey)
	return &AnthropicProvider{
		client:    anthropic.NewClient(requestConfig),
		model:     model,
		maxTokens: maxTokens,
	}
}

func (a *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	// Convert messages
	messages := make([]anthropic.MessageParam, len(req.Messages))
	for i, msg := range req.Messages {
		if msg.Role == "user" {
			messages[i] = anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content))
		} else {
			messages[i] = anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content))
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = a.maxTokens
	}

	response, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(maxTokens),
		Messages:  messages,
	})

	if err != nil {
		return nil, err
	}

	// Extract text content
	var content string
	for _, block := range response.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content:    content,
		TokensUsed: int(response.Usage.InputTokens + response.Usage.OutputTokens),
		Model:      a.model,
		Provider:   "anthropic",
		Latency:    time.Since(start),
		Cached:     false,
	}, nil
}

func (a *AnthropicProvider) Name() string {
	return "anthropic"
}

func (a *AnthropicProvider) HealthCheck(ctx context.Context) error {
	_, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: int64(10),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
		},
	})
	return err
}
