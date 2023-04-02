package generator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Base struct {
	Name        string
	Description string
	Doc         string
}

type WithVars struct {
	RequiredVars []string
}

func (wv *WithVars) ValidateVars(vars map[string]string) error {
	if len(wv.RequiredVars) > 1 {
		if vars == nil {
			return errors.New("vars is required")
		}
		for _, v := range wv.RequiredVars {
			if _, ok := vars[v]; !ok {
				return fmt.Errorf("vars.%q is required", v)
			}
		}
	}
	return nil
}

func (b *Base) GeneratorName() string {
	return b.Name
}

func (b *Base) GeneratorDescription() string {
	return b.Description
}

func (b *Base) GeneratorDoc() string {
	return b.Doc
}

type Generator interface {
	GeneratorName() string
	GeneratorDescription() string
	GeneratorDoc() string
	Generate(path string, vars map[string]string) error
}
type Logger func(generator, path, action string)

type Resolver func(dest, proposed string) error

func replace(dest, origin string) error {
	src, err := os.Open(origin)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		panic(err)
	}

	os.Remove(origin)
	return nil
}

func equalFiles(a, b string) bool {
	cmd := exec.Command("diff", a, b)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd.Run() == nil
}

const GREEN = "\033[32m"
const RED = "\033[31m"
const YELLOW = "\033[33m"
const BLUE = "\033[34m"
const RESET = "\033[0m"

const sucfmt = GREEN + "%10s" + RESET + " %s\n"
const errfmt = RED + "%10s" + RESET + " %s\n"

func Infof(msg string, args ...interface{}) {
	fmt.Print(BLUE + fmt.Sprintf(msg, args...) + RESET)
}
func Log(generator, path, action string) {
	fmt.Printf(sucfmt, action, path)
}

const resolveMsg = `The file %s already exists.
	r - replace
	s - skip
	a - abort
	d - diff
What you want to do? [R/s/a/d]`

func UserResolver(dest, proposed string) error {
	fmt.Printf(errfmt, "conflict", dest)
	b := make([]byte, 1)

	for {
		fmt.Printf(resolveMsg, dest)
		os.Stdin.Read(b)
		switch b[0] {
		case '\n', '\r':
			fallthrough
		case 'r', 'R':
			return replace(dest, proposed)
		case 's', 'S':
			return nil
		case 'a', 'A':
			os.Exit(-2)
		case 'd', 'D':
			exec.Command("diff", dest, proposed).Run()
		}
	}
}

func OverWriteResolver(dest, proposed string) error {
	return replace(dest, proposed)
}

func CompareContent(o Resolver) Resolver {
	return func(dest, proposed string) error {
		if equalFiles(proposed, dest) {
			return nil
		}
		return o(dest, proposed)
	}
}
