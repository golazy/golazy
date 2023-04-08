package lazyaction

import (
	"reflect"
	"regexp"

	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

type Values map[string][]string

var keyformat = regexp.MustCompile(`^([a-zA-Z0-9_]+)\[([a-zA-Z0-9_]+)\](.*)$`)

func (v Values) Extract(key string) Values {
	a := make(Values)
	for k, value := range v {

		matches := keyformat.FindStringSubmatch(k)
		if len(matches) == 0 {
			continue
		}
		newKey := matches[2] + matches[3]
		a[newKey] = value
	}
	return a
}

func (v Values) Load(data any) error {

	t := reflect.TypeOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("data must be a struct")
	}

	err := decoder.Decode(data, v)
	if err != nil {
		return err
	}
	return nil

}
