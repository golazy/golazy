package tty

import (
	"context"
	"errors"
)

// Transport carries terminal-control requests to a server.
type Transport interface {
	RoundTrip(context.Context, Request) (Response, error)
}

// Client sends terminal-control requests to a server.
type Client struct {
	transport Transport
}

func NewClient(transport Transport) (*Client, error) {
	if transport == nil {
		return nil, ErrNilTransport
	}
	return &Client{transport: transport}, nil
}

func (c *Client) IsTerminal(ctx context.Context) (bool, error) {
	resp, err := c.do(ctx, Request{Op: OpIsTerminal})
	if err != nil {
		return false, err
	}
	return resp.IsTerminal, nil
}

func (c *Client) Size(ctx context.Context) (Size, error) {
	resp, err := c.do(ctx, Request{Op: OpSize})
	if err != nil {
		return Size{}, err
	}
	return resp.Size, nil
}

func (c *Client) Resize(ctx context.Context, size Size) error {
	_, err := c.do(ctx, Request{Op: OpResize, Size: size})
	return err
}

func (c *Client) MakeRaw(ctx context.Context) (StateID, error) {
	resp, err := c.do(ctx, Request{Op: OpMakeRaw})
	if err != nil {
		return "", err
	}
	return resp.State, nil
}

func (c *Client) Restore(ctx context.Context, state StateID) error {
	_, err := c.do(ctx, Request{Op: OpRestore, State: state})
	return err
}

func (c *Client) do(ctx context.Context, req Request) (Response, error) {
	if c == nil || c.transport == nil {
		return Response{}, ErrNilTransport
	}
	resp, err := c.transport.RoundTrip(ctx, req)
	if err != nil {
		return resp, err
	}
	if resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}
