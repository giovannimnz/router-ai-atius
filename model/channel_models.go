package model

import (
	"encoding/json"
	"strings"
)

// parseChannelModels normalizes the channel.Models field which can be either:
//   - a JSON array (e.g. ["MiniMax-M3","MiniMax-M2.7"]) — stored as the raw
//     string returned by the PostgreSQL JSON column when GORM reads it.
//   - a CSV string (e.g. "model-a,model-b") — used by older imports.
//
// The previous implementation called strings.Split(",") directly which
// mangled the JSON-array form: a channel with models
//
//	["MiniMax-M3","MiniMax-M2.7"]
//
// was split into `["MiniMax-M3"` and `"MiniMax-M2.7"`, so the in-memory
// channel cache never contained an entry for the bare model name. The
// symptom was a "record not found" 503 from the distributor on any model
// stored in JSON-array form (notably the recently added MiniMax-M3).
func parseChannelModels(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	// JSON array form
	if strings.HasPrefix(trimmed, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
			out := make([]string, 0, len(arr))
			for _, m := range arr {
				m = strings.TrimSpace(m)
				if m != "" {
					out = append(out, m)
				}
			}
			return out
		}
		// fall through to CSV split if JSON parse fails
	}
	// CSV form
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
