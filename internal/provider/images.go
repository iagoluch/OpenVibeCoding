package provider

func openAIMessages(messages []Message) []any {
	out := make([]any, 0, len(messages))
	for _, m := range messages {
		if len(m.Images) == 0 {
			out = append(out, m)
			continue
		}
		parts := []any{map[string]string{"type": "text", "text": m.Content}}
		for _, img := range m.Images {
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64," + img}})
		}
		out = append(out, map[string]any{"role": m.Role, "content": parts})
	}
	return out
}
