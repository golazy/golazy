package apptest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"

	"github.com/gorilla/websocket"
)

func (t *Tester) Connect(ins ...any) (conn *WSConn) {
	t.t.Helper()

	if app, ok := t.app.(app); ok {
		app.Init()
	}

	r := &Request{}
	fillRequest(r, ins...)

	u, err := url.Parse(r.URL)
	if err != nil {
		panic(err)
	}

	conn = &WSConn{
		t:      t.t,
		app:    t.app,
		server: httptest.NewServer(t.app),
	}

	sURL, err := url.Parse(conn.server.URL)
	if err != nil {
		panic(err)
	}
	u.Scheme = sURL.Scheme
	u.Host = sURL.Host

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	url := u.String()

	wsconn, resp, err := websocket.DefaultDialer.Dial(url, r.Headers)
	if err != nil {
		conn.err = err
		t.t.Fatal("Can't connect")
	}

	if resp.StatusCode != 101 {
		conn.err = fmt.Errorf("can't connect: %s", resp.Status)
		t.t.Fatal(conn.err)
	}

	conn.conn = wsconn

	return conn
}

type WSConn struct {
	t      T
	server *httptest.Server
	err    error
	app    http.Handler
	conn   *websocket.Conn
	req    *Request
}

func (c *WSConn) Send(msg []byte) {
	c.t.Helper()

	err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		c.t.Fatal(err)
	}
}

func (c *WSConn) JSend(msg any) {
	c.t.Helper()
	err := c.conn.WriteJSON(msg)
	if err != nil {
		c.t.Fatal(err)
	}
}

func (c *WSConn) SendCommand(name string, args any) {
	c.t.Helper()

	data, err := json.Marshal(args)
	if err != nil {
		c.t.Fatal(err)
	}
	err = c.conn.WriteJSON(map[string]interface{}{
		"command": name,
		"data":    string(data),
	})
	if err != nil {
		c.t.Fatal(err)
	}
}

func (c *WSConn) Close() {
	c.conn.Close()
	c.server.Close()
}

func (c *WSConn) expectString(msg string) {
	//c.t.Helper()

	_, data, err := c.conn.ReadMessage()
	if err != nil {
		c.t.Fatal(err)
	}

	if msg != string(data) {
		c.t.Fatalf("Expected %v, got %v", msg, string(data))
	}
}

func (c *WSConn) expectBytes(msg []byte) {
	c.t.Helper()

	_, data, err := c.conn.ReadMessage()
	if err != nil {
		c.t.Fatal(err)
	}

	if !reflect.DeepEqual(data, msg) {
		c.t.Fatalf("Expected %v, got %v", msg, data)
	}
}

func (c *WSConn) ExpectCommand(name string, args ...any) {
	//c.t.Helper()

	var obj any
	err := c.conn.ReadJSON(&obj)
	if err != nil {
		c.t.Fatal(err)
	}

	cmd, ok := obj.(map[string]interface{})
	if !ok {
		c.t.Fatal("Expected command object. Got", obj)
	}

	// Ensure we get the correct command
	if cmd["command"] != name {
		c.t.Fatalf("Expected command %q, got %v: %q", name, cmd["command"], cmd)
	}

	// do we have something to compare?
	if args[0] == nil {
		return
	}

	// Did we get data?
	if cmd["data"] == nil {
		c.t.Fatal("Expected data to be present. Got", obj)
	}

	// Decode the data
	var dataR []byte
	switch data := cmd["data"].(type) {
	case string:
		dataR = []byte(data)
	case []byte:
		dataR = data
	default:
		c.t.Fatalf("Expected data to be a string. Got %T %v", cmd["data"], cmd["data"])
	}

	// Create a new struct of the same type as the expectation
	t := reflect.TypeOf(args[0])
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	newT := reflect.New(t).Interface()

	err = json.Unmarshal(dataR, newT)
	if err != nil {
		c.t.Fatal(err)
	}

	// And compare
	compare(c.t, args[0], newT)

}

func compare(t T, expecation, reply any) {
	expecation = remarshal(expecation)
	reply = remarshal(reply)

	switch expecation := expecation.(type) {
	case []byte:
		if !reflect.DeepEqual(expecation, reply) {
			t.Fatalf("Expected %v, got %v", expecation, reply)
		}
	case string:
		if !reflect.DeepEqual(expecation, reply) {
			t.Fatalf("Expected %v, got %v", expecation, reply)
		}
	case map[string]interface{}:
		for k, v := range expecation {
			compare(t, v, reply.(map[string]interface{})[k])
		}
	case []interface{}:
		for i, v := range expecation {
			compare(t, v, reply.([]interface{})[i])
		}
	default:
		if !reflect.DeepEqual(expecation, reply) {
			t.Fatalf("Expected %v, got %v", expecation, reply)
		}
	}

}

func remarshal(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var obj any
	err = json.Unmarshal(data, &obj)
	if err != nil {
		panic(err)
	}
	return obj
}

func (c *WSConn) Expect(msg any) {
	//c.t.Helper()

	switch msg := msg.(type) {
	case []byte:
		c.expectBytes(msg)
		return
	case string:
		c.expectString(msg)
		return
	}

	obj := reflect.New(reflect.TypeOf(msg)).Interface()
	err := c.conn.ReadJSON(obj)
	if err != nil {
		c.t.Fatal(err)
	}

	if !reflect.DeepEqual(obj, msg) {
		c.t.Fatalf("Expected %v, got %v", msg, obj)
	}
}
