package apptemplate

import "log/slog"

type Logger interface {
	Progress(current, total int)
	Step(string)
	Prompt(string) string
	Error(error)
}

type logger struct {
}

func (l logger) Progress(current, total int) {
	slog.Info("progress", "progress", total/current)
}

func (l logger) Step(s string) {
	slog.Info("STEP: " + s)

}

func (l logger) Error(e error) {
	slog.Info("ERROR: " + e.Error())

}

func (l logger) Prompt(string) string {
	return ""
}

var DefaultLogger Logger = logger{}
