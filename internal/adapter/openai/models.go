package openai

// imageModelIDs is the set of OpenAI model IDs that support image generation.
// Only gpt-image-2 is current; the rest are deprecated.
var imageModelIDs = map[string]bool{
	"gpt-image-2": true,
}

// editCapableModels is the set of image model IDs that support image editing.
var editCapableModels = map[string]bool{
	"gpt-image-2": true,
}
