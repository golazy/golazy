package tty

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSize(t *testing.T) {
	tests := map[string]struct {
		size  Size
		valid bool
		text  string
	}{
		"valid":     {Size{Rows: 24, Cols: 80}, true, "24x80"},
		"zero rows": {Size{Rows: 0, Cols: 80}, false, "0x80"},
		"zero cols": {Size{Rows: 24, Cols: 0}, false, "24x0"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := test.size.Valid(); got != test.valid {
				t.Fatalf("Valid() = %v, want %v", got, test.valid)
			}
			if got := test.size.String(); got != test.text {
				t.Fatalf("String() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestDeviceDelegatesToBackend(t *testing.T) {
	backend := &fakeBackend{terminal: true, size: Size{Rows: 10, Cols: 20}}
	device, err := NewDevice(backend)
	if err != nil {
		t.Fatalf("NewDevice returned error: %v", err)
	}

	if !device.IsTerminal() {
		t.Fatal("expected terminal backend")
	}

	size, err := device.Size()
	if err != nil {
		t.Fatalf("Size returned error: %v", err)
	}
	if size != (Size{Rows: 10, Cols: 20}) {
		t.Fatalf("Size = %v", size)
	}

	if err := device.Resize(Size{Rows: 30, Cols: 100}); err != nil {
		t.Fatalf("Resize returned error: %v", err)
	}
	if backend.size != (Size{Rows: 30, Cols: 100}) {
		t.Fatalf("backend size = %v", backend.size)
	}
}

func TestDeviceRejectsInvalidResize(t *testing.T) {
	device, err := NewDevice(&fakeBackend{})
	if err != nil {
		t.Fatalf("NewDevice returned error: %v", err)
	}

	if err := device.Resize(Size{}); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("Resize error = %v, want ErrInvalidSize", err)
	}
}

func TestNilDeviceMethods(t *testing.T) {
	var device *Device

	if device.IsTerminal() {
		t.Fatal("nil device reported terminal")
	}
	if _, err := device.Size(); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("Size error = %v, want ErrNilBackend", err)
	}
	if err := device.Resize(Size{Rows: 1, Cols: 1}); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("Resize error = %v, want ErrNilBackend", err)
	}
	if _, err := device.MakeRaw(); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("MakeRaw error = %v, want ErrNilBackend", err)
	}
	if err := device.Restore(newState("raw")); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("Restore error = %v, want ErrNilBackend", err)
	}
}

func TestClientServerRoundTrip(t *testing.T) {
	backend := &fakeBackend{terminal: true, size: Size{Rows: 24, Cols: 80}}
	client := newTestClient(t, backend)
	ctx := context.Background()

	terminal, err := client.IsTerminal(ctx)
	if err != nil {
		t.Fatalf("IsTerminal returned error: %v", err)
	}
	if !terminal {
		t.Fatal("expected terminal")
	}

	size, err := client.Size(ctx)
	if err != nil {
		t.Fatalf("Size returned error: %v", err)
	}
	if size != (Size{Rows: 24, Cols: 80}) {
		t.Fatalf("Size = %v", size)
	}

	if err := client.Resize(ctx, Size{Rows: 40, Cols: 120}); err != nil {
		t.Fatalf("Resize returned error: %v", err)
	}
	if backend.size != (Size{Rows: 40, Cols: 120}) {
		t.Fatalf("backend size = %v", backend.size)
	}
}

func TestClientServerRawRestore(t *testing.T) {
	backend := &fakeBackend{}
	client := newTestClient(t, backend)
	ctx := context.Background()

	id, err := client.MakeRaw(ctx)
	if err != nil {
		t.Fatalf("MakeRaw returned error: %v", err)
	}
	if id == "" {
		t.Fatal("expected state id")
	}
	if backend.rawCalls != 1 {
		t.Fatalf("raw calls = %d", backend.rawCalls)
	}

	if err := client.Restore(ctx, id); err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if backend.restoreCalls != 1 {
		t.Fatalf("restore calls = %d", backend.restoreCalls)
	}

	if err := client.Restore(ctx, id); err == nil {
		t.Fatal("expected second restore to fail")
	}
}

func TestClientServerCanceledContext(t *testing.T) {
	client := newTestClient(t, &fakeBackend{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Size(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Size error = %v, want context.Canceled", err)
	}
}

func TestClientTransportErrors(t *testing.T) {
	sentinel := errors.New("transport failed")
	client, err := NewClient(roundTripFunc(func(context.Context, Request) (Response, error) {
		return Response{}, sentinel
	}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	terminal, err := client.IsTerminal(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("IsTerminal error = %v, want sentinel", err)
	}
	if terminal {
		t.Fatal("IsTerminal returned true with transport error")
	}

	client, err = NewClient(roundTripFunc(func(context.Context, Request) (Response, error) {
		return errorResponse(sentinel), nil
	}))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if _, err := client.MakeRaw(context.Background()); err == nil || err.Error() != sentinel.Error() {
		t.Fatalf("MakeRaw error = %v, want %q", err, sentinel.Error())
	}

	var nilClient *Client
	if _, err := nilClient.Size(context.Background()); !errors.Is(err, ErrNilTransport) {
		t.Fatalf("nil client Size error = %v, want ErrNilTransport", err)
	}
}

func TestServerErrorBranches(t *testing.T) {
	ctx := context.Background()

	var nilServer *Server
	if resp := nilServer.Serve(ctx, Request{Op: OpSize}); !strings.Contains(resp.Error, ErrNilBackend.Error()) {
		t.Fatalf("nil server response = %#v, want ErrNilBackend", resp)
	}

	server := newTestServer(t, &fakeBackend{sizeErr: errors.New("size failed")})
	if resp := server.Serve(ctx, Request{Op: OpSize}); resp.Error != "size failed" {
		t.Fatalf("size response = %#v", resp)
	}

	server = newTestServer(t, &fakeBackend{})
	if resp := server.Serve(ctx, Request{Op: OpResize}); !strings.Contains(resp.Error, ErrInvalidSize.Error()) {
		t.Fatalf("resize response = %#v, want ErrInvalidSize", resp)
	}
	if resp := server.Serve(ctx, Request{Op: "unknown"}); resp.Error == "" {
		t.Fatal("expected unknown operation error")
	}

	server = newTestServer(t, &fakeBackend{rawErr: errors.New("raw failed")})
	if resp := server.Serve(ctx, Request{Op: OpMakeRaw}); resp.Error != "raw failed" {
		t.Fatalf("raw response = %#v", resp)
	}

	backend := &fakeBackend{restoreErr: errors.New("restore failed")}
	client := newTestClient(t, backend)
	id, err := client.MakeRaw(ctx)
	if err != nil {
		t.Fatalf("MakeRaw returned error: %v", err)
	}
	if err := client.Restore(ctx, id); err == nil || err.Error() != "restore failed" {
		t.Fatalf("Restore error = %v, want restore failed", err)
	}
}

func TestMessageEncoding(t *testing.T) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)

	req := Request{Op: OpResize, Size: Size{Rows: 30, Cols: 100}}
	resp := Response{State: "state-1", IsTerminal: true}

	if err := encoder.EncodeRequest(req); err != nil {
		t.Fatalf("EncodeRequest returned error: %v", err)
	}
	if err := encoder.EncodeResponse(resp); err != nil {
		t.Fatalf("EncodeResponse returned error: %v", err)
	}

	decoder := NewDecoder(&buf)
	gotReq, err := decoder.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest returned error: %v", err)
	}
	if gotReq != req {
		t.Fatalf("request = %#v, want %#v", gotReq, req)
	}

	gotResp, err := decoder.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse returned error: %v", err)
	}
	if gotResp != resp {
		t.Fatalf("response = %#v, want %#v", gotResp, resp)
	}
}

func TestMessageDecodeErrors(t *testing.T) {
	decoder := NewDecoder(strings.NewReader("{"))
	if _, err := decoder.DecodeRequest(); err == nil {
		t.Fatal("expected DecodeRequest error")
	}

	decoder = NewDecoder(strings.NewReader("{"))
	if _, err := decoder.DecodeResponse(); err == nil {
		t.Fatal("expected DecodeResponse error")
	}
}

func TestNilInputs(t *testing.T) {
	if _, err := NewDevice(nil); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("NewDevice error = %v, want ErrNilBackend", err)
	}
	if _, err := NewClient(nil); !errors.Is(err, ErrNilTransport) {
		t.Fatalf("NewClient error = %v, want ErrNilTransport", err)
	}
	if _, err := NewServer(nil); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("NewServer error = %v, want ErrNilBackend", err)
	}
	if _, err := Open(nil); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("Open error = %v, want ErrNilBackend", err)
	}
}

func newTestClient(t *testing.T, backend Backend) *Client {
	t.Helper()

	device, err := NewDevice(backend)
	if err != nil {
		t.Fatalf("NewDevice returned error: %v", err)
	}
	server, err := NewServer(device)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	client, err := NewClient(server)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	return client
}

func newTestServer(t *testing.T, backend Backend) *Server {
	t.Helper()

	device, err := NewDevice(backend)
	if err != nil {
		t.Fatalf("NewDevice returned error: %v", err)
	}
	server, err := NewServer(device)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	return server
}

type fakeBackend struct {
	terminal bool
	size     Size

	rawCalls     int
	restoreCalls int

	sizeErr    error
	resizeErr  error
	rawErr     error
	restoreErr error
}

func (f *fakeBackend) IsTerminal() bool {
	return f.terminal
}

func (f *fakeBackend) Size() (Size, error) {
	if f.sizeErr != nil {
		return Size{}, f.sizeErr
	}
	return f.size, nil
}

func (f *fakeBackend) Resize(size Size) error {
	if !size.Valid() {
		return ErrInvalidSize
	}
	if f.resizeErr != nil {
		return f.resizeErr
	}
	f.size = size
	return nil
}

func (f *fakeBackend) MakeRaw() (*State, error) {
	if f.rawErr != nil {
		return nil, f.rawErr
	}
	f.rawCalls++
	return newState("raw"), nil
}

func (f *fakeBackend) Restore(state *State) error {
	if state == nil || state.value != "raw" {
		return ErrUnknownState
	}
	if f.restoreErr != nil {
		return f.restoreErr
	}
	f.restoreCalls++
	return nil
}

type roundTripFunc func(context.Context, Request) (Response, error)

func (f roundTripFunc) RoundTrip(ctx context.Context, req Request) (Response, error) {
	return f(ctx, req)
}
