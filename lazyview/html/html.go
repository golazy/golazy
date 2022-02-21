/* Package html have all the html elements and attributes */
package html

import (
	"github.com/guillermo/golazy/lazyview/nodes"
)

//go:generate ./generate_tags
//go:generate ./generate_attr

// DataAttr sets a data-* attribute.
func DataAttr(attr string, value ...string) nodes.Attr {
	return nodes.NewAttr("data-"+attr, value...)
}
