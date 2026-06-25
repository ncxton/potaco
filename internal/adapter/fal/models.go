package fal

import (
	"strings"

	"github.com/ncxton/potaco/internal/adapter"
)

var editEndpointSuffixes = []string{"image-to-image", "edit"}

func isEditEndpoint(modelID string) bool {
	for _, suffix := range editEndpointSuffixes {
		if strings.Contains(modelID, suffix) {
			return true
		}
	}
	return false
}

var fallbackModels = []adapter.Model{
	{ID: "fal-ai/flux/dev", DisplayName: "Flux Dev", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"guidance_scale", "num_inference_steps", "seed", "output_format", "image_size", "num_images", "enable_safety_checker"}},
	{ID: "fal-ai/flux/schnell", DisplayName: "Flux Schnell", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"num_inference_steps", "seed", "output_format", "image_size", "num_images"}},
	{ID: "fal-ai/nano-banana", DisplayName: "Nano Banana", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"aspect_ratio", "output_format", "safety_tolerance", "system_prompt"}},
}

var hardcodedModelParams = map[string][]adapter.Param{
	"fal-ai/flux/": {
		{Name: "guidance_scale", Type: "float", Description: "Guidance scale for generation", Default: "3.5"},
		{Name: "num_inference_steps", Type: "int", Description: "Number of inference steps", Default: "50"},
		{Name: "seed", Type: "int", Description: "Reproducibility seed", Default: "0"},
		{Name: "output_format", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg", "webp"}},
		{Name: "image_size", Type: "string", Description: "Image dimensions (WxH or preset)", Default: "1024x1024"},
		{Name: "num_images", Type: "int", Description: "Number of images", Default: "1"},
		{Name: "enable_safety_checker", Type: "bool", Description: "Enable safety checker", Default: "true"},
	},
	"fal-ai/nano-banana": {
		{Name: "aspect_ratio", Type: "string", Description: "Aspect ratio (e.g., 16:9, 1:1)", Default: "1:1"},
		{Name: "output_format", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg"}},
		{Name: "safety_tolerance", Type: "int", Description: "Safety tolerance level (1-6)", Default: "2"},
		{Name: "system_prompt", Type: "string", Description: "System prompt", Default: ""},
	},
}

func lookupModelParams(modelID string) ([]adapter.Param, bool) {
	for prefix, params := range hardcodedModelParams {
		if strings.HasPrefix(modelID, prefix) {
			return params, true
		}
	}
	return nil, false
}
