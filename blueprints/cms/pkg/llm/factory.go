package llm

import "fmt"

// NewClient creates a new LLM client based on the configuration.
func NewClient(config *Config) (Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch config.Provider {
	case "openai", "":
		return NewOpenAIClient(config), nil
	case "anthropic":
		return NewAnthropicClient(config), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// MustNewClient creates a new LLM client or panics.
func MustNewClient(config *Config) Client {
	client, err := NewClient(config)
	if err != nil {
		panic(err)
	}
	return client
}
