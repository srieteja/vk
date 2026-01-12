package providers

import (
	"fmt"
	"os"
)

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	// ProviderTypeAnthropic represents the Anthropic provider
	ProviderTypeAnthropic ProviderType = "anthropic"
	// ProviderTypeOpenAI represents the OpenAI provider
	ProviderTypeOpenAI ProviderType = "openai"
)

// Config holds the configuration for creating a provider
type Config struct {
	Type      ProviderType
	APIKey    string
	Model     string
	MaxTokens int
}

// NewProvider creates a new LLM provider based on the provided configuration
func NewProvider(config Config) (Provider, error) {
	switch config.Type {
	case ProviderTypeAnthropic:
		return NewAnthropicProvider(config.APIKey, config.Model, config.MaxTokens), nil
	case ProviderTypeOpenAI:
		return NewOpenAIProvider(config.APIKey, config.Model, config.MaxTokens), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// NewProviderFromEnv creates a new LLM provider using environment variables
func NewProviderFromEnv(providerType ProviderType) (Provider, error) {
	switch providerType {
	case ProviderTypeAnthropic:
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		return NewAnthropicProvider(apiKey, "claude-2", 1000), nil

	case ProviderTypeOpenAI:
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		return NewOpenAIProvider(apiKey, "gpt-4", 1000), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
