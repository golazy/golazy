package html

import (
	"encoding/json"
	"io"

	"golazy.dev/lazyml"
)

func JSON(data any) io.WriterTo {

	out, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		return lazyml.Raw(err.Error())
	}
	return Pre(Code(StyleAttr("font-size: 8px;line-height: 10px; white-space: pre-wrap;"), lazyml.Raw(string(out))))

}
