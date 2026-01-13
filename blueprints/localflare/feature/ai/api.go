// Package ai provides Workers AI management.
package ai

import (
	"context"
	"errors"
)

// Errors
var (
	ErrModelNotFound = errors.New("model not found")
)

// Model represents an AI model.
type Model struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Task        string                 `json:"task"`
	Properties  map[string]interface{} `json:"properties"`
}

// InferenceResult represents inference results.
type InferenceResult struct {
	Result interface{} `json:"result"`
	Usage  *Usage      `json:"usage,omitempty"`
}

// Usage tracks token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// RunModelIn contains input for running a model.
type RunModelIn struct {
	Inputs  map[string]interface{} `json:"inputs"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// API defines the AI service contract.
type API interface {
	ListModels(ctx context.Context, task string) ([]*Model, error)
	GetModel(ctx context.Context, name string) (*Model, error)
	RunModel(ctx context.Context, modelName string, in *RunModelIn) (*InferenceResult, error)
}

// Store defines the data access contract.
type Store interface {
	ListModels(ctx context.Context, task string) ([]*Model, error)
	GetModel(ctx context.Context, name string) (*Model, error)
	Run(ctx context.Context, model string, inputs map[string]interface{}, options map[string]interface{}) (*InferenceResult, error)
}
