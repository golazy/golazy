package lazysupport

import (
	"fmt"
	"html/template"
	"strings"
)

func SimpleFormat(s any) string {
	return strings.ReplaceAll(
		template.HTMLEscapeString(fmt.Sprint(s)),
		"\n", "<br>")
}
