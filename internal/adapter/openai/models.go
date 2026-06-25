package openai

import "github.com/ncxton/potaco/internal/adapter"

// imageModelIDs is the set of OpenAI model IDs that support image generation.
var imageModelIDs = map[string]bool{
	"gpt-image-2":      true,
	"gpt-image-1":      true,
	"gpt-image-1-mini": true,
	"dall-e-3":         true,
	"dall-e-2":         true,
}

// editCapableModels is the set of image model IDs that support image editing.
var editCapableModels = map[string]bool{
	"gpt-image-2":      true,
	"gpt-image-1":      true,
	"gpt-image-1-mini": true,
	"dall-e-2":         true,
}

// fallbackModels is the hardcoded list used when API discovery fails.
var fallbackModels = []adapter.Model{
	{ID: "gpt-image-2", DisplayName: "GPT Image 2", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n", "background", "output_format", "output_compression", "moderation"}},
	{ID: "gpt-image-1", DisplayName: "GPT Image 1", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n"}},
	{ID: "gpt-image-1-mini", DisplayName: "GPT Image 1 Mini", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n"}},
	{ID: "dall-e-3", DisplayName: "DALL-E 3", SupportsGen: true, SupportsEdit: false, Capabilities: []string{"size", "quality", "style", "n"}},
	{ID: "dall-e-2", DisplayName: "DALL-E 2", SupportsGen: true, SupportsEdit: true, Capabilities: []string{"size", "quality", "n"}},
}

// hardcodedModelParams maps model IDs to their supported parameters.
var hardcodedModelParams = map[string][]adapter.Param{
	"gpt-image-2": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536", "auto"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
		{Name: "background", Type: "enum", Description: "Background type", Default: "auto", EnumValues: []string{"transparent", "opaque", "auto"}},
		{Name: "output_format", Type: "enum", Description: "Output format", Default: "png", EnumValues: []string{"png", "jpeg", "webp"}},
		{Name: "output_compression", Type: "int", Description: "Output compression (0-100)", Default: "0"},
		{Name: "moderation", Type: "enum", Description: "Moderation level", Default: "auto", EnumValues: []string{"auto", "low"}},
	},
	"gpt-image-1": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
	},
	"gpt-image-1-mini": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1536x1024", "1024x1536"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "auto", EnumValues: []string{"auto", "low", "medium", "high"}},
		{Name: "n", Type: "int", Description: "Number of images", Default: "1"},
	},
	"dall-e-3": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"1024x1024", "1792x1024", "1024x1792"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "standard", EnumValues: []string{"standard", "hd"}},
		{Name: "style", Type: "enum", Description: "Visual style", Default: "vivid", EnumValues: []string{"vivid", "natural"}},
		{Name: "n", Type: "int", Description: "Number of images (always 1 for dall-e-3)", Default: "1"},
	},
	"dall-e-2": {
		{Name: "size", Type: "enum", Description: "Image dimensions", Default: "1024x1024", EnumValues: []string{"256x256", "512x512", "1024x1024"}},
		{Name: "quality", Type: "enum", Description: "Image quality", Default: "standard", EnumValues: []string{"standard"}},
		{Name: "n", Type: "int", Description: "Number of images (1-10)", Default: "1"},
	},
}
