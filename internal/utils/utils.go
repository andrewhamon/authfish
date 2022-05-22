package utils

import (
	"strings"
)

func NormalizeUsername(username string) string {
	return strings.TrimSpace(
		username,
	)
}
