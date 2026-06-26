package openai

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
