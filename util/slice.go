package util

import "strings"

func InSlice(str string, values []string) bool {
	for _, v := range values {
		if str == v {
			return true
		}
	}
	return false
}

func InSliceContains(str string, values []string) bool {
	for _, v := range values {
		if strings.Contains(v, str) {
			return true
		}
	}
	return false
}
