package devserver

import (
	"fmt"
	"path/filepath"

	"golazy.dev/lazydev/filewatcher"
)

func watch(path string) (changes <-chan (filewatcher.ChangeSet), close func(), err error) {

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get absolute path: %w.\nKilling and building again", err)
	}

	fw, err := filewatcher.New(abs)
	if err != nil {
		return nil, nil, fmt.Errorf("can't start the file watcher: %w.\nKilling and building again", err)
	}
	changes, err = fw.Watch()
	if err != nil {
		return nil, nil, fmt.Errorf("can't watch directory: %w.\nKilling and building again", err)
	}
	close = func() { fw.Close() }
	return

}
