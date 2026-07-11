package tty

import (
	"encoding/json"
	"io"
)

// Op identifies a terminal-control request.
type Op string

const (
	OpIsTerminal Op = "is_terminal"
	OpSize       Op = "size"
	OpResize     Op = "resize"
	OpMakeRaw    Op = "make_raw"
	OpRestore    Op = "restore"
)

// Request is a wire-friendly terminal-control request.
type Request struct {
	Op    Op      `json:"op"`
	Size  Size    `json:"size"`
	State StateID `json:"state,omitempty"`
}

// Response is a wire-friendly terminal-control response.
type Response struct {
	Size       Size    `json:"size"`
	State      StateID `json:"state,omitempty"`
	IsTerminal bool    `json:"is_terminal,omitempty"`
	Error      string  `json:"error,omitempty"`
}

func errorResponse(err error) Response {
	if err == nil {
		return Response{}
	}
	return Response{Error: err.Error()}
}

// Encoder writes newline-delimited request/response messages.
type Encoder struct {
	encoder *json.Encoder
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{encoder: json.NewEncoder(w)}
}

func (e *Encoder) EncodeRequest(req Request) error {
	return e.encoder.Encode(req)
}

func (e *Encoder) EncodeResponse(resp Response) error {
	return e.encoder.Encode(resp)
}

// Decoder reads newline-delimited request/response messages.
type Decoder struct {
	decoder *json.Decoder
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{decoder: json.NewDecoder(r)}
}

func (d *Decoder) DecodeRequest() (Request, error) {
	var req Request
	err := d.decoder.Decode(&req)
	return req, err
}

func (d *Decoder) DecodeResponse() (Response, error) {
	var resp Response
	err := d.decoder.Decode(&resp)
	return resp, err
}
