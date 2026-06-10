package harness

import "strings"

func containsInsensitive(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
