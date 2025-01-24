package head

import (
	"testing"
)

// Test if the Script struct is a Component.
var _ Component = &Script{}

func TestScript(t *testing.T) {

	tt := tester{t}
	tt.expect(Script{}, "")
	tt.expect(Script{Src: "test.js"}, `<script src=test.js type=module></script>`)
	tt.expect(Script{Content: "console.log('test')"}, "<script type=module>\nconsole.log('test')\n</script>")

}
