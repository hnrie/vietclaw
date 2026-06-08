package providers

import "strings"

const maxProviderErrorLength = 300

func SanitizeError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxProviderErrorLength {
		return value[:maxProviderErrorLength]
	}
	return value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
