package providers

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type CustomProvider struct {
	client *openai.Client
	model  string
}

// NewCustomProvider creates a provider that points to a custom OpenAI-compatible endpoint
// (e.g., Ollama, vLLM, LocalAI)
func NewCustomProvider(apiKey string, baseURL string, model string) *CustomProvider {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL

	return &CustomProvider{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}
}

func (p *CustomProvider) Name() string {
	return "custom-local"
}

func (p *CustomProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Convert internal messages to OpenAI messages
	var messages []openai.ChatCompletionMessage
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       p.model,
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Temperature: float32(req.Temperature),
			User:        req.UserId,
		},
	)

	if err != nil {
		return nil, err
	}

	return &CompletionResponse{
		Content:    resp.Choices[0].Message.Content,
		TokensUsed: resp.Usage.TotalTokens,
		Model:      p.model,
		Provider:   "custom",
		Cached:     false,
	}, nil
}

func (p *CustomProvider) HealthCheck(ctx context.Context) error {
	// Simple model list check to verify connection
	_, err := p.client.ListModels(ctx)
	return err
}
