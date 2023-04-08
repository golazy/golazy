package lazysupport

func Truncate(text string, length int) string {
	if len(text) > length {
		return text[:length] + "..."
	}
	return text
}
