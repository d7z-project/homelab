package networkcommon

import "strings"

func LooksSensitiveConfigKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(key, "password") ||
		strings.Contains(key, "passwd") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "credential") ||
		strings.Contains(key, "api_key") ||
		strings.Contains(key, "apikey") ||
		strings.Contains(key, "auth") ||
		strings.Contains(key, "username") ||
		strings.Contains(key, "user") ||
		strings.Contains(key, "pass")
}
