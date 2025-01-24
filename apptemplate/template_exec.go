package apptemplate

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
)

func (t *Template) Exec(cmd string) {
	t.actions = append(t.actions, &execAction{
		cmd: cmd,
	})

}

type execAction struct {
	cmd string
}

func render(input string, vars any) (string, error) {
	t, err := template.New("template").Parse(input)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, vars)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (a *execAction) Run(ctx runCtx) {

	command, err := render(a.cmd, ctx.Vars)
	if err != nil {
		panic(err)
	}
	args := []string{"-c", command}
	//args = append(args, strings.Split(command, " ")...)

	// Ensure cmd.Dir exists in go
	err = os.MkdirAll(ctx.Dest, 0755)
	if err != nil {
		panic(err)
	}

	fmt.Println("Running command:", args[1:len(args)])
	cmd := exec.Command("/bin/bash", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Dir = ctx.Dest

	err = cmd.Run()
	if err != nil {
		panic(err)
	}

}

func (a *execAction) Name() string {
	return fmt.Sprintf("run %q", a.cmd)
}

func (a *execAction) Steps() int {
	return 1
}
