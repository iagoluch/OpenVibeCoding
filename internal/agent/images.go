package agent

import (
	"encoding/base64"
	"encoding/json"
)

// Screenshots are transmitted for the active turn, never persisted in history.
func (e *Engine) captureImages(result string) string {
	var value any
	if json.Unmarshal([]byte(result), &value) != nil {
		return result
	}
	var images []string
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if image, ok := x["image"].(string); ok {
				if b, err := base64.StdEncoding.DecodeString(image); err == nil && len(b) > 8 && len(b) < 6<<20 && string(b[:8]) == "\x89PNG\r\n\x1a\n" {
					if len(images) < 3 {
						images = append(images, image)
					}
				}
				x["image"] = "[transient screenshot; not retained in session]"
			}
			for _, v := range x {
				walk(v)
			}
		case []any:
			for _, v := range x {
				walk(v)
			}
		}
	}
	walk(value)
	if len(images) > 0 {
		e.images = images
	}
	b, _ := json.Marshal(value)
	return string(b)
}
