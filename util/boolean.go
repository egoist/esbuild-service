package util

import "strings"

func StrToBool(key string) bool {
	if strings.ToLower(key) == "true" {
		return true
	}
	return false
}
