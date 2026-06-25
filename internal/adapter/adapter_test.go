// internal/adapter/adapter_test.go
package adapter

import (
	"context"
	"errors"
	"testing"
)

func TestAdapterInterfaceCompile(t *testing.T) {
	// Compile-time check that a minimal struct implements Adapter
	var _ Adapter = &mockAdapter{}
}

type mockAdapter struct{}

func (m *mockAdapter) Name() string { return "mock" }
func (m *mockAdapter) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{}, nil
}
func (m *mockAdapter) Edit(ctx context.Context, req EditRequest) (*GenerateResponse, error) {
	return &GenerateResponse{}, nil
}
func (m *mockAdapter) DiscoverModels(ctx context.Context) ([]Model, error) { return nil, nil }
func (m *mockAdapter) Verify(ctx context.Context) error                    { return nil }
func (m *mockAdapter) ModelParams(ctx context.Context, modelID string) ([]Param, error) {
	return nil, nil
}
func (m *mockAdapter) AuthHeader(apiKey string) string { return "Bearer " + apiKey }

func TestAdapterErrors(t *testing.T) {
	if !errors.Is(ErrEditNotSupported, ErrEditNotSupported) {
		t.Error("ErrEditNotSupported should be a sentinel error")
	}
	if !errors.Is(ErrModelNotFound, ErrModelNotFound) {
		t.Error("ErrModelNotFound should be a sentinel error")
	}
	if !errors.Is(ErrVerificationFailed, ErrVerificationFailed) {
		t.Error("ErrVerificationFailed should be a sentinel error")
	}
	if !errors.Is(ErrDiscoveryFailed, ErrDiscoveryFailed) {
		t.Error("ErrDiscoveryFailed should be a sentinel error")
	}
}

func TestGenerateRequestFields(t *testing.T) {
	req := GenerateRequest{
		Prompt:         "a cat",
		Model:          "gpt-image-2",
		N:              1,
		Size:           "1024x1024",
		Quality:        "auto",
		Style:          "vivid",
		ResponseFormat: "b64_json",
		Seed:           42,
		GuidanceScale:  7.5,
		NegativePrompt: "blurry",
		ExtraParams:    map[string]any{"background": "transparent"},
	}
	if req.Prompt != "a cat" {
		t.Errorf("Prompt = %q", req.Prompt)
	}
	if req.ExtraParams["background"] != "transparent" {
		t.Errorf("ExtraParams not set correctly")
	}
}

func TestEditRequestFields(t *testing.T) {
	req := EditRequest{
		Prompt:      "make it blue",
		Model:       "gpt-image-2",
		ImagePath:   "/tmp/test.png",
		MaskPath:    "/tmp/mask.png",
		N:           1,
		Size:        "1024x1024",
		ExtraParams: map[string]any{"strength": 0.8},
	}
	if req.ImagePath != "/tmp/test.png" {
		t.Errorf("ImagePath = %q", req.ImagePath)
	}
	if req.ExtraParams["strength"] != 0.8 {
		t.Errorf("ExtraParams not set correctly")
	}
}

func TestGenerateResponseFields(t *testing.T) {
	resp := GenerateResponse{
		Created: 1234567890,
		Data: []ImageData{
			{B64JSON: "aGVsbG8=", URL: "", RevisedPrompt: "a fluffy cat"},
		},
	}
	if resp.Data[0].B64JSON != "aGVsbG8=" {
		t.Errorf("B64JSON = %q", resp.Data[0].B64JSON)
	}
	if resp.Data[0].RevisedPrompt != "a fluffy cat" {
		t.Errorf("RevisedPrompt = %q", resp.Data[0].RevisedPrompt)
	}
}

func TestModelFields(t *testing.T) {
	m := Model{
		ID:           "gpt-image-2",
		DisplayName:  "GPT Image 2",
		SupportsGen:  true,
		SupportsEdit: true,
		Capabilities: []string{"quality", "background", "output_format"},
	}
	if !m.SupportsGen {
		t.Error("SupportsGen should be true")
	}
	if !m.SupportsEdit {
		t.Error("SupportsEdit should be true")
	}
	if len(m.Capabilities) != 3 {
		t.Errorf("Capabilities len = %d, want 3", len(m.Capabilities))
	}
}

func TestParamFields(t *testing.T) {
	p := Param{
		Name:        "size",
		Type:        "enum",
		Description: "Image dimensions",
		Default:     "1024x1024",
		EnumValues:  []string{"1024x1024", "1536x1024", "1024x1536"},
		Required:    false,
	}
	if p.Default != "1024x1024" {
		t.Errorf("Default = %q", p.Default)
	}
	if len(p.EnumValues) != 3 {
		t.Errorf("EnumValues len = %d, want 3", len(p.EnumValues))
	}
}
