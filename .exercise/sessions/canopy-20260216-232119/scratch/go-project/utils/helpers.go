package utils

import "strings"

// FormatName formats a user name
func FormatName(first, last string) string {
	return strings.TrimSpace(first) + " " + strings.TrimSpace(last)
}

// ValidateEmail checks if an email is valid
func ValidateEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

var defaultDomain = "example.com"

func getDefaultEmail(name string) string {
	return strings.ToLower(name) + "@" + defaultDomain
}
