package values

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/gorilla/schema"
)

type Values map[string][]string

var keyformat = regexp.MustCompile(`^([a-zA-Z0-9_]*)\[([a-zA-Z0-9_]*)\](.*)$`)

func (v Values) ExtractList(keys ...string) map[string]Values {
	target := v.Extract(keys...)

	out := map[string]Values{}

	for k, value := range target {
		i := strings.IndexByte(k, '[')
		if i == -1 {
			continue
		}
		id := k[0:i]

		if entry, ok := out[id]; ok {
			entry[k] = value
		} else {
			out[id] = Values{k: value}
		}
	}

	for k, v := range out {
		entry := v.Extract(k)
		if _, ok := entry["_destroy"]; ok {
			delete(out, k)
			continue
		}
		out[k] = entry
	}
	return out
}

func (v Values) Extract(keys ...string) Values {
	if len(keys) == 0 {
		return v
	}
	key := keys[0]

	a := make(Values)
	for k, value := range v {

		matches := keyformat.FindStringSubmatch(k)
		if len(matches) < 2 || matches[1] != key {
			continue
		}
		newKey := matches[2] + matches[3]
		a[newKey] = value
	}
	return a.Extract(keys[1:]...)
}

func (v Values) Load(data any) error {

	t := reflect.TypeOf(data)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("data must be a struct")
	}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)

	err := decoder.Decode(data, v)
	if err != nil {
		return err
	}
	return nil

}
