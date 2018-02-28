package util

func InSlice(str string, values []string) bool {
	for _, v := range values {
		if str == v {
			return true
		}
	}
	return false
}
