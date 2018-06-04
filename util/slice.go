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

func InSliceContains(subStr string, values []string) bool {
	for _, v := range values {
		if strings.Contains(v, subStr) {
			return true
		}
	}
	return false
}
