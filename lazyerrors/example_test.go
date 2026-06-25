package lazyerrors_test

import (
	"errors"
	"fmt"
	"strings"

	"golazy.dev/lazyerrors"
)

func ExampleNew() {
	err := loadPost("hello")

	fmt.Println(err)
	fmt.Println(errors.Is(err, errNotFound))
	fmt.Println(strings.Contains(backtraceOf(err)[0].String(), "loadPost"))

	// Output:
	// lazyerrors_test.loadPost: load post "hello": not found
	// true
	// true
}

var errNotFound = errors.New("not found")

//go:noinline
func loadPost(id string) error {
	return lazyerrors.New("load post %q: %w", id, errNotFound)
}

func backtraceOf(err error) []lazyerrors.Frame {
	var traced interface {
		Backtrace() []lazyerrors.Frame
	}
	if !errors.As(err, &traced) {
		return nil
	}
	return traced.Backtrace()
}
