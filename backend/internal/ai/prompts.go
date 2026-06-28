package ai

import (
	"embed"
	"strings"
)

//go:embed prompts/*.md
var promptFS embed.FS

// Prompt templates are loaded from docs/ai-workflow/prompts/ at init time.
// This keeps prompt versions independent of Go source versions (per ADR-0005).
var (
	promptAttribution   string
	promptCompletion    string
	promptPrioritization string
)

func init() {
	load := func(name string) string {
		data, err := promptFS.ReadFile("prompts/" + name)
		if err != nil {
			// Fall back to minimal prompt — the real ones live in docs/.
			return "You are an AI assistant. Respond with valid JSON."
		}
		return strings.TrimSpace(string(data))
	}
	promptAttribution = load("attribution.md")
	promptCompletion = load("completion.md")
	promptPrioritization = load("priority.md")
}
