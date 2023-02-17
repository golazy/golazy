package lazylog

import (
	"strings"
)

type Logger interface {
	Log(Message)
}

type Messager interface {
	Message() Message
}

type TerminalLogger struct {
}

func (l *TerminalLogger) Log(m Message) {

}

type Message map[string]string

func NewMessage(args ...interface{}) Message {
	m := make(Message)
	for _, item := range args {
		switch item := item.(type) {
		case string:
			if strings.Contains(item, " ") || !strings.Contains(item, "=") {
				m["message"] = item
				continue
			}
			parts := strings.SplitN(item, "=", 2)
			m[parts[0]] = parts[1]
		case Messager:
			for k, v := range item.Message() {
				m[k] = v
			}
		default:
			panic(item)
		}
	}
	return m
}
