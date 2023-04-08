/* Package html have all the html elements and attributes */
package html

import (
	"golazy.dev/lazyview/nodes"
)

//go:generate ./generate_tags
//go:generate ./generate_attr

// DataAttr sets a data-* attribute.
func DataAttr(attr string, value ...string) nodes.Attr {
	return nodes.NewAttr("data-"+attr, value...)
}

func DataAttrs(values map[string]string) []nodes.Attr {
	attrs := make([]nodes.Attr, len(values))

	for k, v := range values {
		attrs = append(attrs, DataAttr(k, v))
	}
	return attrs
}

func Attribute(attr string, value ...string) nodes.Attr {
	return nodes.NewAttr(attr, value...)
}
