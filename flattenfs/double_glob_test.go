package flattenfs

import (
	"fmt"
	"regexp"
	"testing"
)

func TestToRegExp(t *testing.T) {

	expect := func(in, out string) {
		t.Run(fmt.Sprintf("%q", in), func(t *testing.T) {

			reg := toRegexp(in)
			if reg != out {
				t.Error("Expected", in, "to generate", out, "Got", reg)
			}
			_, err := regexp.Compile(reg)
			if err != nil {
				t.Error("Invalid regexp", reg, err)
			}
		})
	}

	expect("**/", "^.*$")
	expect("a", "^a$")
	expect(".*", `^\.[^/]*$`)
	expect("**/*.tpl", `^.*[^/]*\.tpl$`)

}

func TestDoubleGlob(t *testing.T) {

	files, err := Glob(FS, "**/*.tpl")
	if err != nil {
		t.Fatal(err)
	}

	includes := func(file string) {
		for _, f := range files {
			if f == file {
				return
			}
		}
		t.Error("file not found", file, "Have", fmt.Sprint(files))
	}

	includes("test1/test2/test3/3.tpl")
	includes("test1/1.tpl")
}
