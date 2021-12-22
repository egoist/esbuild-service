package util

import "strings"

func StrToBool(key string) bool {
	return strings.ToLower(key) == "true"
}
