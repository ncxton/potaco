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
	SupportsGenerate() bool
	SupportsEdit() bool
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error)
	DiscoverModels(ctx context.Context) ([]Model, error)
	Verify(ctx context.Context) error
	AuthHeader(apiKey string) string
}

// GenerateRequest is the normalized request for image generation.
// Provider-specific fields pass through ExtraParams.
type GenerateRequest struct {
	Prompt         string         `json:"prompt"`
	Model          string         `json:"model"`
	N              int            `json:"n"`
	Size           string         `json:"size"`
	Quality        string         `json:"quality"`
	Style          string         `json:"style"`
	ResponseFormat string         `json:"response_format"`
	Seed           int            `json:"seed"`
	GuidanceScale  float64        `json:"guidance_scale"`
	NegativePrompt string         `json:"negative_prompt"`
	ExtraParams    map[string]any `json:"extra_params,omitempty"`
}

// EditRequest is the normalized request for image editing.
// Provider-specific fields pass through ExtraParams.
type EditRequest struct {
	Prompt         string         `json:"prompt"`
	Model          string         `json:"model"`
	N              int            `json:"n"`
	Size           string         `json:"size"`
	ResponseFormat string         `json:"response_format"`
	ImagePath      string         `json:"image_path"`
	MaskPath       string         `json:"mask_path"`
	User           string         `json:"user"`
	ExtraParams    map[string]any `json:"extra_params,omitempty"`
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

// Sentinel errors for adapter operations.
var (
	ErrEditNotSupported   = errors.New("image editing not supported by this provider")
	ErrVerificationFailed = errors.New("provider verification failed")
	ErrDiscoveryFailed    = errors.New("model discovery failed")
)
