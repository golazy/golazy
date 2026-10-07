package lazymcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStdioDiscoveryAndRequestIDs(t *testing.T) {
	scope := NewScope(Options{Views: &fakeViews{}})
	if err := scope.Register(newAdminMCP(context.Background())); err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":0,"method":"initialize"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"notifications/unknown"}`,
		`{"jsonrpc":"2.0","id":9007199254740993,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":"resources","method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":"app","method":"resources/read","params":{"uri":"ui://admin/dashboard"}}`,
		`{"jsonrpc":"2.0","id":"skill","method":"resources/read","params":{"uri":"skill://admin/oncall/SKILL.md"}}`,
		`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"admin.user_count","arguments":{}}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := scope.ServeStdio(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	replies := decodeReplies(t, output.Bytes())
	if len(replies) != 6 {
		t.Fatalf("received %d replies", len(replies))
	}
	for _, id := range []string{"0", "9007199254740993", `"resources"`, `"app"`, `"skill"`, `"call"`} {
		if replies[id] == nil {
			t.Fatalf("missing ID %s", id)
		}
	}
	call := replies[`"call"`]["result"].(map[string]any)
	if call["_meta"].(map[string]any)["details"] != "widget only" {
		t.Fatal("widget metadata missing")
	}
	if strings.Contains(string(mustJSON(call["structuredContent"])), "widget only") {
		t.Fatal("widget data leaked")
	}
}

func TestStdioAndHTTPShareResults(t *testing.T) {
	for _, method := range []string{"initialize", "tools/list", "resources/list", "prompts/list", "resources/templates/list"} {
		t.Run(method, func(t *testing.T) {
			scope := NewScope(Options{})
			if err := scope.Register(newAdminMCP(context.Background())); err != nil {
				t.Fatal(err)
			}
			payload := `{"jsonrpc":"2.0","id":0,"method":"` + method + `"}`
			response := httptest.NewRecorder()
			scope.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(payload)))
			var output bytes.Buffer
			if err := scope.ServeStdio(context.Background(), strings.NewReader(payload+"\n"), &output); err != nil {
				t.Fatal(err)
			}
			httpReply := decodeReplies(t, response.Body.Bytes())["0"]
			stdioReply := decodeReplies(t, output.Bytes())["0"]
			// Only one module and tool, so map traversal does not reorder arrays.
			if !bytes.Equal(mustJSON(httpReply), mustJSON(stdioReply)) {
				t.Fatalf("HTTP and stdio differ for %s", method)
			}
		})
	}
}

func TestStdioMalformedMessagesAndEOF(t *testing.T) {
	scope := NewScope(Options{})
	for _, tc := range []struct {
		input string
		code  int
	}{
		{"{\n", -32700},
		{"[]\n", -32600},
		{`{"jsonrpc":"1.0","id":7,"method":"ping"}` + "\n", -32600},
		{`{"jsonrpc":"2.0","id":true,"method":"ping"}` + "\n", -32600},
		{`{"jsonrpc":"2.0","id":"x","method":"missing"}` + "\n", -32601},
	} {
		var output bytes.Buffer
		if err := scope.ServeStdio(context.Background(), strings.NewReader(tc.input), &output); err != nil {
			t.Fatal(err)
		}
		var response struct {
			Error rpcError `json:"error"`
		}
		if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &response); err != nil {
			t.Fatal(err)
		}
		if response.Error.Code != tc.code {
			t.Fatalf("code = %d, want %d", response.Error.Code, tc.code)
		}
	}
	var output bytes.Buffer
	if err := scope.ServeStdio(context.Background(), strings.NewReader(""), &output); err != nil || output.Len() != 0 {
		t.Fatalf("EOF: %v", err)
	}
	recorder := httptest.NewRecorder()
	scope.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)))
	if recorder.Code != http.StatusAccepted || recorder.Body.Len() != 0 {
		t.Fatal("HTTP replied to notification")
	}
}

type blockingMCP struct {
	Base
	started chan struct{}
	stopped chan struct{}
}

func (m *blockingMCP) WaitTool(context.Context) ToolSpec { return ToolSpec{Fn: m.Wait} }
func (m *blockingMCP) Wait(ctx context.Context) (map[string]any, error) {
	close(m.started)
	<-ctx.Done()
	close(m.stopped)
	return nil, ctx.Err()
}

func TestStdioCancellationAndConcurrentPing(t *testing.T) {
	scope := NewScope(Options{})
	module := &blockingMCP{Base: NewBase(context.Background()), started: make(chan struct{}), stopped: make(chan struct{})}
	if err := scope.Register(module); err != nil {
		t.Fatal(err)
	}
	input, client := io.Pipe()
	reader, output := io.Pipe()
	defer reader.Close()
	result := make(chan error, 1)
	go func() { result <- scope.ServeStdio(context.Background(), input, output); output.Close() }()
	_, err := io.WriteString(client, `{"jsonrpc":"2.0","id":"slow","method":"tools/call","params":{"name":"blocking.wait"}}`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	awaitSignal(t, module.started)
	go func() {
		io.WriteString(client, `{"jsonrpc":"2.0","id":2,"method":"ping"}`+"\n")
		io.WriteString(client, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":"slow"}}`+"\n")
		client.Close()
	}()
	var response rpcResponse
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if string(response.ID) != "2" {
		t.Fatalf("received %s, want ping response", response.ID)
	}
	awaitSignal(t, module.stopped)
	trailing, err := io.ReadAll(reader)
	if err != nil || len(trailing) != 0 {
		t.Fatalf("cancelled request replied: %s, %v", trailing, err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestStdioContextCancellationUnblocksReadAndRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	scope := NewScope(Options{})
	module := &blockingMCP{Base: NewBase(ctx), started: make(chan struct{}), stopped: make(chan struct{})}
	if err := scope.Register(module); err != nil {
		t.Fatal(err)
	}
	input, client := io.Pipe()
	defer client.Close()
	var output bytes.Buffer
	result := make(chan error, 1)
	go func() { result <- scope.ServeStdio(ctx, input, &output) }()
	io.WriteString(client, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"blocking.wait"}}`+"\n")
	awaitSignal(t, module.started)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not stop server")
	}
	awaitSignal(t, module.stopped)
	if output.Len() != 0 {
		t.Fatal("cancelled request replied")
	}
}

type largeMCP struct{ Base }

func (m *largeMCP) EchoTool(context.Context) ToolSpec { return ToolSpec{Fn: m.Echo} }
func (m *largeMCP) Echo(_ context.Context, p struct {
	Text string `json:"text"`
}) (userCountResult, error) {
	return userCountResult{Count: len(p.Text), Details: p.Text}, nil
}
func TestStdioLargePayloadsAndSerializedResponses(t *testing.T) {
	scope := NewScope(Options{})
	if err := scope.Register(&largeMCP{Base: NewBase(context.Background())}); err != nil {
		t.Fatal(err)
	}
	var input bytes.Buffer
	payload := strings.Repeat("x", 256<<10)
	for id := range 12 {
		json.NewEncoder(&input).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": "large.echo", "arguments": map[string]any{"text": payload}}})
	}
	var output bytes.Buffer
	if err := scope.ServeStdio(context.Background(), &input, &output); err != nil {
		t.Fatal(err)
	}
	replies := decodeReplies(t, output.Bytes())
	if len(replies) != 12 {
		t.Fatalf("received %d replies", len(replies))
	}
	for _, reply := range replies {
		result := reply["result"].(map[string]any)
		if result["_meta"].(map[string]any)["details"] != payload {
			t.Fatal("payload truncated")
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestStdioWriteFailureStopsSession(t *testing.T) {
	err := NewScope(Options{}).ServeStdio(context.Background(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`+"\n"), failingWriter{})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("error = %v", err)
	}
}

func decodeReplies(t *testing.T, data []byte) map[string]map[string]any {
	t.Helper()
	replies := make(map[string]map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var raw map[string]json.RawMessage
		err := decoder.Decode(&raw)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		var reply map[string]any
		if err := json.Unmarshal(mustJSON(raw), &reply); err != nil {
			t.Fatal(err)
		}
		key := string(raw["id"])
		if replies[key] != nil {
			t.Fatalf("duplicate response %s", key)
		}
		replies[key] = reply
	}
	return replies
}
func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

type notifyingWriter struct {
	io.WriteCloser
	started chan struct{}
}

func (writer notifyingWriter) Write(data []byte) (int, error) {
	close(writer.started)
	return writer.WriteCloser.Write(data)
}
func TestStdioContextCancellationUnblocksOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input, client := io.Pipe()
	defer client.Close()
	reader, output := io.Pipe()
	defer reader.Close()
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() { result <- NewScope(Options{}).ServeStdio(ctx, input, notifyingWriter{output, started}) }()
	io.WriteString(client, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`+"\n")
	awaitSignal(t, started)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancellation did not unblock output")
	}
}
