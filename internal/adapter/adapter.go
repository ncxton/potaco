// internal/adapter/adapter.go
package adapter

import (
	"context"
	"errors"
)

// Adapter is the interface that each provider implements to abstract over
// their API differences for image generation, editing, model discovery,
// and verification.
type Adapter interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error)
	DiscoverModels(ctx context.Context) ([]Model, error)
	Verify(ctx context.Context) error
	ModelParams(ctx context.Context, modelID string) ([]Param, error)
	AuthHeader(apiKey string) string
}

// GenerateRequest is the normalized request for image generation.
// Provider-specific fields pass through ExtraParams.
type GenerateRequest struct {
	Prompt         string
	Model          string
	N              int
	Size           string
	Quality        string
	Style          string
	ResponseFormat string
	Seed           int
	GuidanceScale  float64
	NegativePrompt string
	ExtraParams    map[string]any
}

// EditRequest is the normalized request for image editing.
// Provider-specific fields pass through ExtraParams.
type EditRequest struct {
	Prompt         string
	Model          string
	N              int
	Size           string
	ResponseFormat string
	ImagePath      string
	MaskPath       string
	User           string
	ExtraParams    map[string]any
}

// GenerateResponse is the normalized response from both generate and edit.
type GenerateResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData represents a single generated or edited image.
type ImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// Model represents an image-generation-capable model from a provider.
type Model struct {
	ID           string
	DisplayName  string
	SupportsGen  bool
	SupportsEdit bool
	Capabilities []string
}

// Param describes a supported parameter for a specific model.
type Param struct {
	Name        string
	Type        string // "string", "int", "float", "bool", "enum"
	Description string
	Default     string
	EnumValues  []string
	Required    bool
}

// Sentinel errors for adapter operations.
var (
	ErrEditNotSupported   = errors.New("image editing not supported by this provider")
	ErrModelNotFound      = errors.New("model not found")
	ErrVerificationFailed = errors.New("provider verification failed")
	ErrDiscoveryFailed    = errors.New("model discovery failed")
)
