package log

import "log"

type Logger struct {
	ExcludeList []string
}

var DefaultLogger = &Logger{}

func (l *Logger) Println(v ...interface{}) {
	log.Println(v...)
}

func Println(v ...interface{}) {
	DefaultLogger.Println(v...)
}
