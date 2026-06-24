package provider

// GenerateRequest is the JSON body for POST /v1/images/generations.
type GenerateRequest struct {
	Prompt         string  `json:"prompt"`
	Model          string  `json:"model,omitempty"`
	N              int     `json:"n,omitempty"`
	Size           string  `json:"size,omitempty"`
	Quality        string  `json:"quality,omitempty"`
	Style          string  `json:"style,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Seed           int     `json:"seed,omitempty"`
	GuidanceScale  float64 `json:"guidance_scale,omitempty"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	User           string  `json:"user,omitempty"`
}

// EditRequest carries the parameters for POST /v1/images/edits.
// The image and mask are file paths; the client handles encoding them
// into multipart form data.
type EditRequest struct {
	Prompt         string
	Model          string
	N              int
	Size           string
	ResponseFormat string
	ImagePath      string
	MaskPath       string
	User           string
}

// ImageResponse is the JSON response from both endpoints.
type ImageResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// ImageData represents a single generated/edited image.
type ImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ErrorResponse is the JSON error shape returned by OpenAI-compatible APIs.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError holds the details of an API error.
type APIError struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Param   string `json:"param,omitempty"`
}
