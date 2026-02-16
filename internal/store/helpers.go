package store

import (
	"encoding/json"
	"strings"
)

// placeholderList returns "?,?,?" for n placeholders.
func placeholderList(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// int64sToArgs converts []int64 to []any for use with database/sql.
func int64sToArgs(ids []int64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

// repeatArgs repeats args n times (for queries with multiple IN clauses).
func repeatArgs(args []any, n int) []any {
	result := make([]any, 0, len(args)*n)
	for range n {
		result = append(result, args...)
	}
	return result
}

// countSubstring counts non-overlapping occurrences of substr in s.
func countSubstring(s, substr string) int {
	return strings.Count(s, substr)
}

// marshalModifiers converts []string to JSON text for storage.
func marshalModifiers(mods []string) string {
	if len(mods) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(mods)
	return string(b)
}

// UnmarshalModifiers converts JSON text back to []string.
// Exported for use by QueryBuilder.
func UnmarshalModifiers(s string) []string {
	return unmarshalModifiers(s)
}

// unmarshalModifiers converts JSON text back to []string.
func unmarshalModifiers(s string) []string {
	if s == "" || s == "null" {
		return nil
	}
	var mods []string
	_ = json.Unmarshal([]byte(s), &mods)
	return mods
}
