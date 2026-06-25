package vercel

import (
	"strings"

	"github.com/ncxton/potaco/internal/adapter"
)

var fallbackModels = []adapter.Model{
	{ID: "openai/gpt-image-2", DisplayName: "gpt-image-2", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"size", "quality", "n"}},
	{ID: "openai/dall-e-3", DisplayName: "dall-e-3", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"size", "quality", "style", "n"}},
	{ID: "bfl/flux-2-pro", DisplayName: "flux-2-pro", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"outputFormat", "aspectRatio"}},
}

var hardcodedModelParams = map[string][]adapter.Param{
	"openai": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536", "auto"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
	},
	"bfl": {
		{Name: "outputFormat", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg", "webp"}},
		{Name: "aspectRatio", Type: "string", Description: "Aspect ratio", Default: "1:1"},
	},
}

func stripProviderPrefix(modelID string) string {
	if idx := strings.Index(modelID, "/"); idx >= 0 {
		return modelID[idx+1:]
	}
	return modelID
}

func providerPrefix(modelID string) string {
	if idx := strings.Index(modelID, "/"); idx >= 0 {
		return modelID[:idx]
	}
	return ""
}

func lookupModelParams(modelID string) ([]adapter.Param, bool) {
	prefix := providerPrefix(modelID)
	if prefix == "" {
		return nil, false
	}
	params, ok := hardcodedModelParams[prefix]
	return params, ok
}

func modelCapabilities(modelID string) []string {
	params, ok := lookupModelParams(modelID)
	if !ok {
		return nil
	}

	capabilities := make([]string, len(params))
	for i, param := range params {
		capabilities[i] = param.Name
	}
	return capabilities
}
