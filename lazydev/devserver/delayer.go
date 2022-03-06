package devserver

import (
	"strings"
	"time"

	"gopkg.in/fsnotify.v1"
)

const delay = 300

func shouldIgnore(file string) bool {
	return strings.Contains(file, "FETCH_HEAD")
}

func NewDelayer(input <-chan (fsnotify.Event)) <-chan ([]fsnotify.Event) {

	out := make(chan ([]fsnotify.Event))

	go func() {
	Loop:
		for {
			data := make([]fsnotify.Event, 0)
			msg, ok := <-input
			if !ok {
				close(out)
				return
			}
			if shouldIgnore(msg.Name) {
				continue
			}

			data = append(data, msg)
			t := time.After(time.Millisecond * delay)

			for {
				select {
				case msg, ok := <-input:
					if !ok {
						out <- data
						close(out)
						return
					}
					data = append(data, msg)
				case <-t:
					out <- data
					continue Loop
				}
			}
		}
	}()

	return out

}
