package util

import "strings"

func StringSplitArg(stringToSplit, separator string) []interface{} {
	split := strings.Split(stringToSplit, separator)
	splitInterface := make([]interface{}, len(split))
	for i, s := range split {
		splitInterface[i] = s
	}
	return splitInterface
}
