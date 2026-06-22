package lazyassets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"golazy.dev/lazystorage"
)

type UploadOption func(*uploadOptions)

type uploadOptions struct {
	mode          UnpackMode
	prefix        string
	writeManifest bool
}

// WithUploadMode selects which asset paths are written to storage.
func WithUploadMode(mode UnpackMode) UploadOption {
	return func(options *uploadOptions) {
		options.mode = mode
	}
}

// WithUploadPrefix writes assets below prefix in the destination storage.
func WithUploadPrefix(prefix string) UploadOption {
	return func(options *uploadOptions) {
		options.prefix = strings.Trim(prefix, "/")
	}
}

// WithoutUploadManifest disables writing manifest.json to storage.
func WithoutUploadManifest() UploadOption {
	return func(options *uploadOptions) {
		options.writeManifest = false
	}
}

// Upload writes registered assets to object storage.
//
// The default mode writes only permanent content-hashed paths plus manifest.json,
// which is the usual shape for CDN or static-file ingress deployments.
func (r *Registry) Upload(ctx context.Context, storage lazystorage.Writer, options ...UploadOption) error {
	if r == nil {
		return fmt.Errorf("lazyassets: registry is nil")
	}
	if storage == nil {
		return fmt.Errorf("lazyassets: storage writer is nil")
	}
	opts := uploadOptions{mode: UnpackPermanent, writeManifest: true}
	for _, option := range options {
		option(&opts)
	}
	for _, asset := range r.assets {
		if opts.mode != UnpackPermanent {
			if err := r.uploadAsset(ctx, storage, opts.prefix, asset.Path, asset, false); err != nil {
				return err
			}
		}
		if opts.mode != UnpackLogical && asset.Permanent != "" {
			if err := r.uploadAsset(ctx, storage, opts.prefix, asset.Permanent, asset, true); err != nil {
				return err
			}
		}
	}
	if opts.writeManifest {
		data, err := json.MarshalIndent(r.Manifest(), "", "  ")
		if err != nil {
			return fmt.Errorf("marshal manifest: %w", err)
		}
		key := path.Join(opts.prefix, "manifest.json")
		_, _, err = storage.Put(ctx, key, bytes.NewReader(append(data, '\n')),
			lazystorage.ContentType{Value: "application/json"},
			lazystorage.CacheControl{Value: "public, max-age=0, must-revalidate"},
		)
		if err != nil {
			return fmt.Errorf("upload %s: %w", key, err)
		}
	}
	return nil
}

func (r *Registry) uploadAsset(ctx context.Context, storage lazystorage.Writer, prefix, assetPath string, asset *Asset, permanent bool) error {
	reader, err := asset.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", assetPath, err)
	}
	defer reader.Close()
	key := path.Join(prefix, strings.TrimPrefix(assetPath, "/"))
	cache := string(r.options.LogicalCache)
	if permanent {
		cache = string(r.options.PermanentCache)
	}
	var body io.Reader = reader
	_, _, err = storage.Put(ctx, key, body,
		lazystorage.ContentType{Value: asset.ContentType},
		lazystorage.CacheControl{Value: cache},
	)
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}
