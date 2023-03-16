package wscontroller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
)

func newWrapper(obj any) wrapper {
	w := wrapper{
		t: reflect.TypeOf(obj),
	}

	if w.t.Kind() != reflect.Struct &&
		(w.t.Kind() == reflect.Ptr && w.t.Elem().Kind() != reflect.Struct) {
		panic(fmt.Errorf("%s must be a struct", w.t.String()))
	}

	return w
}

type conn struct {
	c *Client
	i any
}

type wrapper struct {
	t      reflect.Type
	connsL sync.Mutex
	conns  []*conn
}

type in struct {
	t  string
	fn func() any
}
type out struct {
	t string
	v any
}

type errNonMethod string

func (e errNonMethod) Error() string {
	return string(e)
}
func (e errNonMethod) Is(a error) bool {
	_, ok := a.(errNonMethod)
	return ok
}

type errMissingArg string

func (e errMissingArg) Error() string {
	return string(e)
}

func (e errMissingArg) Is(a error) bool {
	_, ok := a.(errMissingArg)
	return ok
}

func (w wrapper) call(method string, msg *WSMessage, ins ...in) (outs []out, err error) {
	m, ok := w.t.MethodByName(method)
	if !ok {
		s := fmt.Sprintf("%s does not have a method %s", w.t.String(), method)
		err := errNonMethod(s)
		return nil, err
	}

	fnType := m.Type

	// Fill args
	args := make([]reflect.Value, fnType.NumIn())
	args[0] = w.instance
	for i := 1; i < fnType.NumIn(); i++ {
		in := fnType.In(i)
		t := in.String()
		for _, in := range ins {
			if in.t == t {
				args[i] = reflect.ValueOf(in.fn())
				break
			}
		}
		if args[i].IsValid() {
			continue
		}

		if msg == nil {
			s := fmt.Sprintf("%s %s() expects an argument type %s", w.t.String(), method, in.String())
			err := errMissingArg(s)
			return nil, err
		}

		obj := in
		if obj.Kind() == reflect.Ptr {
			obj = obj.Elem()
		}
		if obj.Kind() != reflect.Struct {
			s := fmt.Sprintf("%s %s() expects an argument type %s", w.t.String(), method, in.String())
			err := errMissingArg(s)
			return nil, err
		}

		paramMsg := reflect.New(obj).Interface()
		err := json.Unmarshal([]byte(msg.Data), paramMsg)
		if err != nil {
			return nil, err
		}

		args[i] = reflect.ValueOf(paramMsg)

	}

	// Call
	rets := m.Func.Call(args)

	// Fill outs
	outs = make([]out, len(rets))
	for i, ret := range rets {
		outs[i] = out{
			t: ret.Type().String(),
			v: ret.Interface(),
		}
	}
	return outs, nil
}

type WSMessage struct {
	Command    string `json:"command"`
	Identifier string `json:"identifier"`
	Data       string `json:"data"`
}

func (c wrapper) Each(func(c *Client, controller any)) {

}
func (c wrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
	}

	var instance reflect.Value
	if c.t.Kind() == reflect.Ptr {
		instance = reflect.New(c.t.Elem())
	} else {
		instance = reflect.New(c.t)
	}

	client := &Client{c: conn, r: r}

	go func() {
		_, err = c.call("Connect", nil,
			in{t: "*wscontroller.Client", fn: func() any { return client }},
			in{t: "*http.Request", fn: func() any { return r }},
		)
		if errors.Is(err, errMissingArg("")) {
			panic(err)
		}

		defer c.call("Disconnect", nil,
			in{t: "*wscontroller.Client", fn: func() any { return client }},
			in{t: "*http.Request", fn: func() any { return r }},
			in{t: "error", fn: func() any { return err }},
		)

		for {
			var msg WSMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				return
			}
			_, err = c.call(msg.Command, &msg,
				in{t: "*wscontroller.Client", fn: func() any { return client }},
				in{t: "*http.Request", fn: func() any { return r }},
			)
			if err != nil {

				data, err := json.Marshal(err.Error())
				if err != nil {
					panic(err)
				}
				client.Send(WSMessage{Command: "error", Data: string(data)})
			}
		}

	}()
}
