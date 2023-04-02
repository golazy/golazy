package html

import (
	"encoding/json"
	"io"

	"golazy.dev/lazyview/nodes"
)

func JSON(data any) io.WriterTo {

	out, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		return nodes.Raw(err.Error())
	}
	return Pre(Code(StyleAttr("font-size: 8px;line-height: 10px; white-space: pre-wrap;"), nodes.Raw(string(out))))

}
