package utils

func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
