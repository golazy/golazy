package lazymcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ServeStdio serves newline-delimited MCP JSON-RPC on input and output.
// Register modules before serving. Responses may arrive out of order; writes
// are serialized, and notifications never receive responses. No logs are
// written to output. Incoming messages are limited to 64 MiB; outgoing messages
// have no transport size limit.
//
// On EOF, ServeStdio finishes pending requests. Cancellation cancels requests
// and closes streams implementing io.Closer to unblock reads and writes. For
// other streams, the caller must unblock I/O before cancellation can finish.
// Streams stay open on normal EOF. ServeStdio returns ctx.Err() on cancellation.
func (scope *Scope) ServeStdio(ctx context.Context, input io.Reader, output io.Writer) error {
	if input == nil || output == nil {
		return errors.New("lazymcp: stdio input and output are required")
	}
	session, cancel := context.WithCancel(ctx)
	defer cancel()
	stopClose := context.AfterFunc(session, func() {
		if closer, ok := input.(io.Closer); ok {
			_ = closer.Close()
		}
		if closer, ok := output.(io.Closer); ok {
			_ = closer.Close()
		}
	})
	defer stopClose()

	frames := make(chan []byte)
	readDone := make(chan error, 1)
	go func() {
		defer close(frames)
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 4096), 64<<20)
		for scanner.Scan() {
			frame := bytes.Clone(scanner.Bytes())
			select {
			case frames <- frame:
			case <-session.Done():
				readDone <- session.Err()
				return
			}
		}
		readDone <- scanner.Err()
	}()

	var writers sync.Mutex
	writeErrors := make(chan error, 1)
	write := func(response rpcResponse) {
		writers.Lock()
		defer writers.Unlock()
		if session.Err() != nil {
			return
		}
		if err := json.NewEncoder(output).Encode(response); err != nil {
			select {
			case writeErrors <- err:
			default:
			}
			cancel()
		}
	}
	var requests sync.Mutex
	pending := make(map[string]context.CancelFunc)
	var workers sync.WaitGroup
	var readErr error
	for frame := range frames {
		if session.Err() != nil {
			break
		}
		request, failure := decodeRPC(frame)
		if failure != nil {
			write(*failure)
			continue
		}
		if len(request.ID) == 0 {
			if request.Method == "notifications/cancelled" {
				var params struct {
					RequestID json.RawMessage `json:"requestId"`
				}
				if json.Unmarshal(request.Params, &params) == nil && len(params.RequestID) != 0 && validRPCID(params.RequestID) {
					requests.Lock()
					if stop := pending[rpcIDKey(params.RequestID)]; stop != nil {
						stop()
					}
					requests.Unlock()
				}
			}
			continue
		}
		// Complete initialization before dispatching subsequent requests. It is
		// deliberately excluded from the cancellable request registry.
		if request.Method == "initialize" {
			write(scope.dispatchRPC(session, request))
			continue
		}
		requestContext, stop := context.WithCancel(session)
		key := rpcIDKey(request.ID)
		requests.Lock()
		if _, duplicate := pending[key]; duplicate {
			requests.Unlock()
			stop()
			write(rpcFailure(request.ID, -32600, "Request ID is already in use"))
			continue
		}
		pending[key] = stop
		requests.Unlock()
		workers.Add(1)
		go func() {
			defer workers.Done()
			defer stop()
			response := scope.dispatchRPC(requestContext, request)
			requests.Lock()
			delete(pending, key)
			cancelled := requestContext.Err() != nil
			requests.Unlock()
			if !cancelled {
				write(response)
			}
		}()
	}
	readErr = <-readDone
	if readErr != nil {
		cancel()
	}
	workers.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	select {
	case err := <-writeErrors:
		return fmt.Errorf("lazymcp: write stdio: %w", err)
	default:
	}
	if readErr != nil {
		return fmt.Errorf("lazymcp: read stdio: %w", readErr)
	}
	return nil
}

func rpcIDKey(id json.RawMessage) string {
	if bytes.Equal(bytes.TrimSpace(id), []byte("null")) {
		return "null"
	}
	var text string
	if json.Unmarshal(id, &text) == nil {
		return "string:" + text
	}
	return "number:" + string(bytes.TrimSpace(id))
}
