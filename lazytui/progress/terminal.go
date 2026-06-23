package progress

import (
	"io"
	"os"
)

var getenv = os.Getenv

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func supportsColor(interactive bool) bool {
	if !interactive {
		return false
	}
	if getenv("NO_COLOR") != "" {
		return false
	}
	if getenv("TERM") == "dumb" {
		return false
	}
	return true
}
