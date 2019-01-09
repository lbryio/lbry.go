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

// SubstringInSlice returns true if str is contained within any element of the values slice. False otherwise
func SubstringInSlice(str string, values []string) bool {
	for _, v := range values {
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}
