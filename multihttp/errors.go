package multihttp

import (
	"errors"
	"sync"
)

func eachError(err error, fn func(err error)) {
	eSlice, ok := err.(interface{ Unwrap() []error })
	if !ok {
		fn(err)
		return
	}
	for _, err := range eSlice.Unwrap() {
		eachError(err, fn)
	}
}

func collectErrors(fns ...func() error) error {
	var errs []error
	var l sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(fns))

	for _, f := range fns {
		go func(f func() error) {
			err := f()
			l.Lock()
			errs = append(errs, err)
			l.Unlock()
			wg.Done()
		}(f)
	}
	wg.Wait()
	return errors.Join(errs...)
}

func collectErrorsSync(fns ...func() error) error {
	var errs []error

	func() {
		for _, f := range fns {
			err := f()
			errs = append(errs, err)
		}
	}()
	return errors.Join(errs...)
}

func filterError(err error, e error) error {
	var errs []error
	eachError(err, func(err error) {
		if !errors.Is(err, e) {
			errs = append(errs, err)
		}
	})
	return errors.Join(errs...)
}
