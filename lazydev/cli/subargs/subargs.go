package subargs

import "os"

var Args = []string{}

func init() {
	for i, arg := range os.Args {
		if arg == "--" {
			Args = os.Args[i+1:]
			os.Args = os.Args[:i]
			return
		}
	}
}
