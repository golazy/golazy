// Package s3 provides an S3-compatible lazystorage backend.
package s3

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"golazy.dev/lazystorage"
)

const (
	service         = "s3"
	unsignedPayload = "UNSIGNED-PAYLOAD"
)

// Storage stores objects in an S3-compatible bucket.
type Storage struct {
	endpoint        string
	region          string
	bucket          string
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	publicBaseURL   string
	client          *http.Client
}

// Option configures Storage.
type Option func(*Storage)

// WithEndpoint sets the S3 API endpoint.
func WithEndpoint(endpoint string) Option {
	return func(storage *Storage) {
		storage.endpoint = strings.TrimRight(endpoint, "/")
	}
}

// WithRegion sets the SigV4 region. S3-compatible stores commonly accept
// us-east-1 even when they are not hosted in AWS.
func WithRegion(region string) Option {
	return func(storage *Storage) {
		storage.region = strings.TrimSpace(region)
	}
}

// WithBucket sets the bucket name.
func WithBucket(bucket string) Option {
	return func(storage *Storage) {
		storage.bucket = strings.Trim(bucket, "/")
	}
}

// WithCredentials sets the access key pair used for signed S3 requests.
func WithCredentials(accessKeyID, secretAccessKey string) Option {
	return func(storage *Storage) {
		storage.accessKeyID = strings.TrimSpace(accessKeyID)
		storage.secretAccessKey = secretAccessKey
	}
}

// WithSessionToken sets the optional session token for temporary credentials.
func WithSessionToken(token string) Option {
	return func(storage *Storage) {
		storage.sessionToken = strings.TrimSpace(token)
	}
}

// WithPublicBaseURL sets the URL prefix returned by URL.
//
// For SeaweedFS deployments this is usually an ingress path that rewrites to
// the filer, for example https://example.com/assets.
func WithPublicBaseURL(baseURL string) Option {
	return func(storage *Storage) {
		storage.publicBaseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient sets the HTTP client used for S3 requests.
func WithHTTPClient(client *http.Client) Option {
	return func(storage *Storage) {
		if client != nil {
			storage.client = client
		}
	}
}

// New creates an S3-compatible storage.
func New(options ...Option) *Storage {
	storage := &Storage{
		region: "us-east-1",
		client: http.DefaultClient,
	}
	for _, option := range options {
		option(storage)
	}
	return storage
}

// Open opens key for reading.
func (s *Storage) Open(ctx context.Context, key string, options ...any) (lazystorage.File, []any, error) {
	if err := lazystorage.ValidateKey(key); err != nil {
		return nil, options, err
	}
	req, err := s.newRequest(ctx, http.MethodGet, key, nil, nil)
	if err != nil {
		return nil, options, err
	}
	resp, err := s.do(req, http.StatusOK)
	if err != nil {
		return nil, options, err
	}
	return &objectFile{ReadCloser: resp.Body, key: key, header: resp.Header}, options, nil
}

// Put writes key to the bucket.
func (s *Storage) Put(ctx context.Context, key string, body io.Reader, options ...any) (lazystorage.Info, []any, error) {
	if err := lazystorage.ValidateKey(key); err != nil {
		return lazystorage.Info{}, options, err
	}
	if body == nil {
		return lazystorage.Info{}, options, fmt.Errorf("lazystorage/s3: nil body")
	}
	contentType, remaining, _ := lazystorage.Take[lazystorage.ContentType](options)
	cacheControl, remaining, _ := lazystorage.Take[lazystorage.CacheControl](remaining)
	disposition, remaining, _ := lazystorage.Take[lazystorage.ContentDisposition](remaining)

	header := http.Header{}
	if contentType.Value != "" {
		header.Set("Content-Type", contentType.Value)
	}
	if cacheControl.Value != "" {
		header.Set("Cache-Control", cacheControl.Value)
	}
	if disposition.Value != "" {
		header.Set("Content-Disposition", disposition.Value)
	}
	header.Set("X-Amz-Content-Sha256", unsignedPayload)

	tracked := &hashingReader{reader: body, hash: sha256.New()}
	req, err := s.newRequest(ctx, http.MethodPut, key, tracked, header)
	if err != nil {
		return lazystorage.Info{}, remaining, err
	}
	resp, err := s.do(req, http.StatusOK)
	if err != nil {
		return lazystorage.Info{}, remaining, err
	}
	_ = resp.Body.Close()

	return lazystorage.Info{
		Key:         key,
		ContentType: contentType.Value,
		Size:        tracked.size,
		Checksum:    "sha256:" + hex.EncodeToString(tracked.hash.Sum(nil)),
		ModifiedAt:  time.Now().UTC(),
	}, remaining, nil
}

// Delete removes key.
func (s *Storage) Delete(ctx context.Context, key string, options ...any) ([]any, error) {
	if err := lazystorage.ValidateKey(key); err != nil {
		return options, err
	}
	req, err := s.newRequest(ctx, http.MethodDelete, key, nil, nil)
	if err != nil {
		return options, err
	}
	resp, err := s.do(req, http.StatusNoContent, http.StatusOK)
	if err != nil {
		return options, err
	}
	return options, resp.Body.Close()
}

// List lists object metadata below prefix.
func (s *Storage) List(ctx context.Context, prefix string, options ...any) (lazystorage.Iterator, []any, error) {
	if prefix != "" {
		if err := lazystorage.ValidateKey(prefix); err != nil {
			return nil, options, err
		}
	}
	query := url.Values{}
	query.Set("list-type", "2")
	if prefix != "" {
		query.Set("prefix", prefix)
	}
	var infos []lazystorage.Info
	for {
		req, err := s.newRequestWithQuery(ctx, http.MethodGet, "", query, nil, nil)
		if err != nil {
			return nil, options, err
		}
		resp, err := s.do(req, http.StatusOK)
		if err != nil {
			return nil, options, err
		}
		var listed listBucketResult
		err = xml.NewDecoder(resp.Body).Decode(&listed)
		closeErr := resp.Body.Close()
		if err != nil {
			return nil, options, err
		}
		if closeErr != nil {
			return nil, options, closeErr
		}
		for _, item := range listed.Contents {
			infos = append(infos, item.info())
		}
		if !listed.IsTruncated || listed.NextContinuationToken == "" {
			break
		}
		query.Set("continuation-token", listed.NextContinuationToken)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Key < infos[j].Key
	})
	return &sliceIterator{infos: infos}, options, nil
}

// URL returns the configured public mount URL for key.
func (s *Storage) URL(ctx context.Context, key string, options ...any) (lazystorage.URL, []any, error) {
	if err := ctx.Err(); err != nil {
		return lazystorage.URL{}, options, err
	}
	if err := lazystorage.ValidateKey(key); err != nil {
		return lazystorage.URL{}, options, err
	}
	if s.publicBaseURL == "" {
		return lazystorage.URL{}, options, fmt.Errorf("lazystorage/s3: public base URL is not configured")
	}
	return lazystorage.URL{String: s.publicBaseURL + "/" + path.Clean(key), Public: true}, options, nil
}

// Watch polls the bucket and emits put/delete events for key changes.
func (s *Storage) Watch(ctx context.Context, prefix string, options ...any) (lazystorage.Events, []any, error) {
	if prefix != "" {
		if err := lazystorage.ValidateKey(prefix); err != nil {
			return nil, options, err
		}
	}
	return &pollEvents{
		storage:  s,
		prefix:   prefix,
		interval: 2 * time.Second,
		known:    map[string]string{},
	}, options, nil
}

// EnsureBucket creates the configured bucket if it does not already exist.
func (s *Storage) EnsureBucket(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return err
	}
	req, err := s.newBucketRequest(ctx, http.MethodPut, nil)
	if err != nil {
		return err
	}
	resp, err := s.do(req, http.StatusOK, http.StatusConflict)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (s *Storage) newRequest(ctx context.Context, method, key string, body io.Reader, header http.Header) (*http.Request, error) {
	return s.newRequestWithQuery(ctx, method, key, nil, body, header)
}

func (s *Storage) newRequestWithQuery(ctx context.Context, method, key string, query url.Values, body io.Reader, header http.Header) (*http.Request, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if key != "" {
		if err := lazystorage.ValidateKey(key); err != nil {
			return nil, err
		}
	}
	u, err := url.Parse(s.endpoint + "/" + path.Join(s.bucket, key))
	if err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	copyHeader(req.Header, header)
	if body == nil {
		req.Body = nil
	}
	return req, s.sign(req)
}

func (s *Storage) newBucketRequest(ctx context.Context, method string, header http.Header) (*http.Request, error) {
	u, err := url.Parse(s.endpoint + "/" + s.bucket)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	copyHeader(req.Header, header)
	return req, s.sign(req)
}

func (s *Storage) do(req *http.Request, statuses ...int) (*http.Response, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	for _, status := range statuses {
		if resp.StatusCode == status {
			return resp, nil
		}
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return nil, fmt.Errorf("lazystorage/s3: %s %s returned %s: %s", req.Method, req.URL.Path, resp.Status, strings.TrimSpace(string(data)))
}

func (s *Storage) validate() error {
	if s.endpoint == "" {
		return fmt.Errorf("lazystorage/s3: endpoint is required")
	}
	if s.bucket == "" {
		return fmt.Errorf("lazystorage/s3: bucket is required")
	}
	if s.region == "" {
		return fmt.Errorf("lazystorage/s3: region is required")
	}
	if s.accessKeyID == "" || s.secretAccessKey == "" {
		return fmt.Errorf("lazystorage/s3: credentials are required")
	}
	return nil
}

func (s *Storage) sign(req *http.Request) error {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	date := now.Format("20060102")
	payloadHash := req.Header.Get("X-Amz-Content-Sha256")
	if payloadHash == "" {
		payloadHash = hashPayload(req)
	}

	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if s.sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", s.sessionToken)
	}

	signedHeaders, canonicalHeaders := canonicalHeaders(req.Header)
	canonicalRequest := strings.Join([]string{
		req.Method,
		encodePath(req.URL.EscapedPath()),
		canonicalQuery(req.URL.Query()),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := date + "/" + s.region + "/" + service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signingKey := deriveSigningKey(s.secretAccessKey, date, s.region)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+s.accessKeyID+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	return nil
}

type hashingReader struct {
	reader io.Reader
	hash   hash.Hash
	size   int64
}

func (r *hashingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		_, _ = r.hash.Write(p[:n])
		r.size += int64(n)
	}
	return n, err
}

func hashPayload(req *http.Request) string {
	if req.Body == nil || req.GetBody == nil {
		return sha256Hex(nil)
	}
	body, err := req.GetBody()
	if err != nil {
		return sha256Hex(nil)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return sha256Hex(nil)
	}
	return sha256Hex(data)
}

func canonicalHeaders(header http.Header) (string, string) {
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	var signed []string
	var canonical strings.Builder
	for _, key := range keys {
		values := header.Values(key)
		for i, value := range values {
			values[i] = strings.Join(strings.Fields(value), " ")
		}
		sort.Strings(values)
		signed = append(signed, key)
		canonical.WriteString(key)
		canonical.WriteByte(':')
		canonical.WriteString(strings.Join(values, ","))
		canonical.WriteByte('\n')
	}
	return strings.Join(signed, ";"), canonical.String()
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var parts []string
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, value := range items {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.ReplaceAll(strings.Join(parts, "&"), "+", "%20")
}

func encodePath(value string) string {
	if value == "" {
		return "/"
	}
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	escaped := strings.ReplaceAll(url.PathEscape(decoded), "%2F", "/")
	if !strings.HasPrefix(escaped, "/") {
		return "/" + escaped
	}
	return escaped
}

func deriveSigningKey(secret, date, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

type objectFile struct {
	io.ReadCloser
	key    string
	header http.Header
}

func (f *objectFile) Stat() (lazystorage.Info, error) {
	size, _ := strconv.ParseInt(f.header.Get("Content-Length"), 10, 64)
	modified, _ := http.ParseTime(f.header.Get("Last-Modified"))
	return lazystorage.Info{
		Key:         f.key,
		ContentType: f.header.Get("Content-Type"),
		Size:        size,
		Checksum:    strings.Trim(f.header.Get("ETag"), `"`),
		ModifiedAt:  modified,
	}, nil
}

type listBucketResult struct {
	IsTruncated           bool          `xml:"IsTruncated"`
	NextContinuationToken string        `xml:"NextContinuationToken"`
	Contents              []listContent `xml:"Contents"`
}

type listContent struct {
	Key          string `xml:"Key"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
}

func (c listContent) info() lazystorage.Info {
	modified, _ := time.Parse(time.RFC3339, c.LastModified)
	return lazystorage.Info{
		Key:        c.Key,
		Size:       c.Size,
		Checksum:   strings.Trim(c.ETag, `"`),
		ModifiedAt: modified,
	}
}

type sliceIterator struct {
	infos []lazystorage.Info
	index int
}

func (i *sliceIterator) Next() (lazystorage.Info, error) {
	if i.index >= len(i.infos) {
		return lazystorage.Info{}, io.EOF
	}
	info := i.infos[i.index]
	i.index++
	return info, nil
}

func (i *sliceIterator) Close() error {
	return nil
}

type pollEvents struct {
	storage  *Storage
	prefix   string
	interval time.Duration
	known    map[string]string
	queued   []lazystorage.Event
}

func (e *pollEvents) Next(ctx context.Context) (lazystorage.Event, error) {
	for len(e.queued) == 0 {
		if err := e.poll(ctx); err != nil {
			return lazystorage.Event{}, err
		}
		if len(e.queued) > 0 {
			break
		}
		timer := time.NewTimer(e.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return lazystorage.Event{}, ctx.Err()
		case <-timer.C:
		}
	}
	event := e.queued[0]
	e.queued = e.queued[1:]
	return event, nil
}

func (e *pollEvents) Close() error {
	e.queued = nil
	return nil
}

func (e *pollEvents) poll(ctx context.Context) error {
	iterator, _, err := e.storage.List(ctx, e.prefix)
	if err != nil {
		return err
	}
	defer iterator.Close()
	current := map[string]string{}
	for {
		info, err := iterator.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		current[info.Key] = info.Checksum
		if e.known[info.Key] != info.Checksum {
			e.queued = append(e.queued, lazystorage.Event{Key: info.Key, Op: "put"})
		}
	}
	for key := range e.known {
		if _, ok := current[key]; !ok {
			e.queued = append(e.queued, lazystorage.Event{Key: key, Op: "delete"})
		}
	}
	e.known = current
	return nil
}
