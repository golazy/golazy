package lazysupport

import (
	"fmt"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/microcosm-cc/bluemonday"
)

func Html2Markdown(input any) string {

	html := fmt.Sprint(input)

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err == nil {
		return markdown
	}

	html = bluemonday.UGCPolicy().Sanitize(html)
	markdown, err = converter.ConvertString(html)
	if err != nil {
		panic(err)
	}
	return markdown

}
