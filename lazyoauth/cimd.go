package lazyoauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxClientIDMetadataDocumentSize = 1 << 20

type clientIDMetadataDocument struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	ClientURI    string   `json:"client_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}

func (server *Server) clientFromMetadataDocument(ctx context.Context, id string) (Client, error) {
	documentURL, err := server.validateClientIDMetadataDocumentURL(id)
	if err != nil {
		return Client{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, documentURL.String(), nil)
	if err != nil {
		return Client{}, invalidMetadataDocument(err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := server.clientIDMetadataDocumentHTTPClient().Do(request)
	if err != nil {
		return Client{}, invalidMetadataDocument(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Client{}, invalidMetadataDocument(fmt.Errorf("unexpected HTTP status %d", response.StatusCode))
	}

	limited := io.LimitReader(response.Body, maxClientIDMetadataDocumentSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return Client{}, invalidMetadataDocument(err)
	}
	if len(data) > maxClientIDMetadataDocumentSize {
		return Client{}, invalidMetadataDocument(fmt.Errorf("document is larger than %d bytes", maxClientIDMetadataDocumentSize))
	}
	var metadata clientIDMetadataDocument
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Client{}, invalidMetadataDocument(err)
	}
	if metadata.ClientID != id || strings.TrimSpace(metadata.ClientName) == "" || len(metadata.RedirectURIs) == 0 {
		return Client{}, invalidMetadataDocument(errors.New("client_id, client_name, and redirect_uris are required"))
	}
	for _, redirectURI := range metadata.RedirectURIs {
		if !validRedirectURI(redirectURI) {
			return Client{}, invalidMetadataDocument(fmt.Errorf("invalid redirect URI %q", redirectURI))
		}
	}
	return Client{
		ID:           metadata.ClientID,
		Name:         metadata.ClientName,
		RedirectURIs: append([]string(nil), metadata.RedirectURIs...),
		Domain:       clientDomain(firstNonEmpty(metadata.ClientURI, metadata.ClientID), metadata.RedirectURIs[0]),
	}, nil
}

func (server *Server) validateClientIDMetadataDocumentURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Path == "" || parsed.Path == "/" || parsed.User != nil || parsed.Fragment != "" {
		return nil, invalidMetadataDocument(errors.New("client ID must be an HTTPS metadata document URL with a path"))
	}
	for _, allowed := range server.config.ClientIDMetadataDocumentHosts {
		if strings.EqualFold(strings.TrimSpace(allowed), parsed.Host) {
			return parsed, nil
		}
	}
	return nil, invalidMetadataDocument(fmt.Errorf("metadata document host %q is not allowed", parsed.Host))
}

func (server *Server) clientIDMetadataDocumentHTTPClient() *http.Client {
	base := server.config.ClientIDMetadataDocumentHTTPClient
	if base == nil {
		base = http.DefaultClient
	}
	client := *base
	if client.Timeout == 0 {
		client.Timeout = 5 * time.Second
	}
	previousCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if _, err := server.validateClientIDMetadataDocumentURL(request.URL.String()); err != nil {
			return err
		}
		if previousCheckRedirect != nil {
			return previousCheckRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("lazyoauth: too many client ID metadata document redirects")
		}
		return nil
	}
	return &client
}

func validRedirectURI(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func invalidMetadataDocument(err error) error {
	return fmt.Errorf("%w: client ID metadata document: %v", ErrInvalidClient, err)
}
