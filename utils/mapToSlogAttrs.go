package utils

import (
	"log/slog"
)

func MapToSlogAttrs(fields map[string]any) []any {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

