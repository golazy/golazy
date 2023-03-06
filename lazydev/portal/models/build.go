package models

type Build struct {
	Output  []byte
	Success bool
}

var LastBuild = &Build{}

func BuildUpdate(success bool, output []byte) {
	LastBuild.Success = success
	LastBuild.Output = output
}
