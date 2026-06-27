package vercel

import "strings"

func stripProviderPrefix(modelID string) string {
	if idx := strings.Index(modelID, "/"); idx >= 0 {
		return modelID[idx+1:]
	}
	return modelID
}
