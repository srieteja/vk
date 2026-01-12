package providers

import (
	"context"
	"time"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client    *openai.Client
	model     string
	maxTokens int
}

func NewOpenAIProvider(apiKey string, model string, maxTokens int) *OpenAIProvider {
	return &OpenAIProvider{
		client:    openai.NewClient(apiKey),
		model:     model,
		maxTokens: maxTokens,
	}
}

func (o *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	start := time.Now()

	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = o.maxTokens
	}

	response, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     o.model,
		Messages:  messages,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, err
	}

	return &CompletionResponse{
		Content:    response.Choices[0].Message.Content,
		TokensUsed: response.Usage.TotalTokens,
		Model:      o.model,
		Provider:   "openai",
		Latency:    time.Since(start),
		Cached:     false,
	}, nil
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) HealthCheck(ctx context.Context) error {
	_, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: o.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: "test"},
		},
		MaxTokens: 10,
	})

	return err
}
