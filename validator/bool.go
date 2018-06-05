package validator

var (
	truthyValues = []string{"1", "yes", "y", "true"}
	falseyValues = []string{"0", "no", "n", "false"}
)

// todo: consider using strconv.ParseBool instead

func IsTruthy(value string) bool {
	for _, e := range truthyValues {
		if e == value {
			return true
		}
	}
	return false
}

func IsFalsey(value string) bool {
	for _, e := range falseyValues {
		if e == value {
			return true
		}
	}
	return false
}

func IsBoolString(value string) bool {
	return IsTruthy(value) || IsFalsey(value)
}

func GetBoolStringValues() []string {
	return append(truthyValues, falseyValues...)
}
