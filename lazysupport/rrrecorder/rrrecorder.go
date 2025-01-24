package rrrecorder

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"

	"golazy.dev/lazysupport"
)

var (
	ErrNonRecordableRequest = fmt.Errorf("non recordable request")
)

type Request struct {
	URL           string
	RequestBody   []byte
	RequestHeader http.Header
	Method        string
}

func printHeader(h http.Header) string {
	s := []string{}
	for k := range h {
		s = append(s, fmt.Sprintf("%s=%s", k, h.Get(k)))
	}

	sort.Strings(s)

	return strings.Join(s, " ")
}

func (r *Request) String() string {
	return fmt.Sprintf(
		"%-10s: %s\n"+
			"%-10s: %s\n"+
			"%-10s: %s\n"+
			"%-10s: %s\n",
		"URL", r.URL,
		"Method", r.Method,
		"Req Header", printHeader(r.RequestHeader),
		"Req Body", lazysupport.Truncate(string(r.RequestBody), 50),
	)
}

type Response struct {
	ResponseBody   []byte
	Status         int
	ResponseHeader http.Header
}

func (r *Response) String() string {
	return fmt.Sprintf(
		"%-10s: %d\n"+
			"%-10s: %s\n"+
			"%-10s: %s\n",
		"Status", r.Status,
		"Res Header", printHeader(r.ResponseHeader),
		"Res Body", lazysupport.Truncate(string(r.ResponseBody), 50),
	)

}

func (r *Response) Write(data []byte) (int, error) {
	r.ResponseBody = append(r.ResponseBody, data...)
	return len(data), nil
}

func (r *Response) Header() http.Header {
	return r.ResponseHeader
}
func (r *Response) WriteHeader(status int) {
	r.Status = status
}

func (r *Response) Flush() error {
	return nil
}

type RoundTrip struct {
	Request
	Response
}

func (r *RoundTrip) String() string {
	return r.Request.String() + r.Response.String()
}

func IsRecordable(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	mimetype, _, _ := mime.ParseMediaType(accept)
	if mimetype == "text/event-stream" {
		return false
	}

	if r.Header.Get("Upgrade") != "" {
		return false
	}
	return true
}

func RecordRequest(req *http.Request) (*Request, error) {
	if req == nil {
		return nil, fmt.Errorf("request can't be nil")
	}

	if !IsRecordable(req) {
		return nil, ErrNonRecordableRequest
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	r := &Request{
		URL:           req.URL.String(),
		RequestBody:   data,
		RequestHeader: req.Header.Clone(),
		Method:        req.Method,
	}

	req.Body = io.NopCloser(bytes.NewReader(data))

	return r, nil

}

func RecordResponse(res *http.Response) (*Response, error) {
	if res == nil {
		return nil, fmt.Errorf("response can't be nil")
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	r := &Response{
		ResponseBody:   data,
		Status:         res.StatusCode,
		ResponseHeader: res.Header.Clone(),
	}

	res.Body = io.NopCloser(bytes.NewReader(data))

	return r, nil
}

func NewRecorder() *Recorder {
	return &Recorder{
		Response{
			ResponseHeader: make(http.Header),
		},
	}
}

type Recorder struct {
	Response
}

func (res *Response) WriteTo(r http.ResponseWriter) error {
	fmt.Println(string(res.ResponseBody))
	for k, v := range res.ResponseHeader {
		for _, vv := range v {
			r.Header().Add(k, vv)
		}
	}
	r.WriteHeader(res.Status)
	_, err := r.Write(res.ResponseBody)
	return err
}

func ServeTo(w http.ResponseWriter, r *http.Request, h http.HandlerFunc) (*RoundTrip, error) {
	req, err := RecordRequest(r)
	if err == ErrNonRecordableRequest {
		return nil, err
	}
	if err != nil {
		h(w, r)
		return nil, err
	}

	rec := NewRecorder()
	h(rec, r)

	rt := &RoundTrip{
		Request:  *req,
		Response: rec.Response,
	}
	return rt, nil
}
