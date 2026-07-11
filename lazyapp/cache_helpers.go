package lazyapp

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"golazy.dev/lazycache"
	"golazy.dev/lazyturbo"
	"golazy.dev/lazyview"
)

type templateCacheKey string

func cacheHelpers() map[string]any {
	return map[string]any{
		"cache":       lazyview.Helper(cachePartialHelper),
		"cachef":      lazyview.Helper(cacheFullPartialHelper),
		"cache_key":   cacheKeyHelper,
		"turbo_frame": lazyview.Helper(cacheTurboFrameHelper),
	}
}

func cacheKeyHelper(parts ...any) (templateCacheKey, error) {
	key, err := lazycache.Key(parts...)
	if err != nil {
		return "", err
	}
	return templateCacheKey(key), nil
}

func cachePartialHelper(ctx *lazyview.Context, args ...any) (any, error) {
	if len(args) < 2 || len(args) > 3 {
		return lazyview.Fragment{}, fmt.Errorf("lazyapp: cache expects name, partial, and optional data")
	}
	name := args[0]
	partial, ok := args[1].(string)
	if !ok || strings.TrimSpace(partial) == "" {
		return lazyview.Fragment{}, fmt.Errorf("lazyapp: cache partial name must be a string")
	}
	data := ctx.Data
	if len(args) == 3 {
		data = args[2]
	}

	key, err := partialCacheKey(ctx, name, partial)
	if err != nil {
		return lazyview.Fragment{}, err
	}
	body, err := cachedPartialBody(ctx, key, partial, data)
	if err != nil {
		return lazyview.Fragment{}, err
	}
	return lazyview.Fragment{Body: body, ContentType: contentTypeForFormat(ctx.Format)}, nil
}

func cacheFullPartialHelper(ctx *lazyview.Context, args ...any) (any, error) {
	if len(args) < 3 {
		return lazyview.Fragment{}, fmt.Errorf("lazyapp: cachef expects key parts, partial, and data")
	}
	partial, ok := args[len(args)-2].(string)
	if !ok || strings.TrimSpace(partial) == "" {
		return lazyview.Fragment{}, fmt.Errorf("lazyapp: cachef partial name must be a string")
	}
	parts := cacheContextPrefix(ctx)
	parts = append(parts, args[:len(args)-2]...)
	key, err := lazycache.Key(parts...)
	if err != nil {
		return lazyview.Fragment{}, err
	}
	body, err := cachedPartialBody(ctx, key, partial, args[len(args)-1])
	if err != nil {
		return lazyview.Fragment{}, err
	}
	return lazyview.Fragment{Body: body, ContentType: contentTypeForFormat(ctx.Format)}, nil
}

func partialCacheKey(ctx *lazyview.Context, name any, partial string) (string, error) {
	if key, ok := name.(templateCacheKey); ok {
		parts := cacheContextPrefix(ctx)
		parts = append(parts, string(key))
		return lazycache.Key(parts...)
	}
	parts := scopedCachePrefix(ctx, partial)
	parts = append(parts, name)
	return lazycache.Key(parts...)
}

func scopedCachePrefix(ctx *lazyview.Context, tail ...any) []any {
	parts := cacheContextPrefix(ctx)
	if strings.TrimSpace(ctx.Namespace) != "" {
		parts = append(parts, ctx.Namespace)
	}
	parts = append(parts, ctx.Controller, ctx.Action, ctx.Format)
	parts = append(parts, tail...)
	return parts
}

func cacheContextPrefix(ctx *lazyview.Context) []any {
	parts := []any{"build", lazycache.BuildVersionFromContext(ctx.Context)}
	var variants []string
	for _, variant := range ctx.Variants {
		if variant = strings.TrimSpace(variant); variant != "" {
			variants = append(variants, variant)
		}
	}
	if len(variants) > 0 {
		parts = append(parts, "variant", strings.Join(variants, "+"))
	}
	return parts
}

func cachedPartialBody(ctx *lazyview.Context, key string, partial string, data any) (string, error) {
	cache, ok := lazycache.FromContext(ctx.Context)
	if !ok {
		return "", fmt.Errorf("lazyapp: cache is missing from render context")
	}
	if body, err := lazycache.Get[string](cache, key); err == nil {
		return body, nil
	} else if err != nil && !errors.Is(err, lazycache.ErrMiss) {
		return "", err
	}

	body, err := renderPartialBody(ctx, partial, data)
	if err != nil {
		return "", err
	}
	if err := cache.Set(body, key); err != nil {
		return "", err
	}
	return body, nil
}

func renderPartialBody(ctx *lazyview.Context, partial string, data any) (string, error) {
	variables := copyVariables(ctx.Variables)
	if data == nil {
		data = ctx.Data
	}
	if locals, ok := data.(map[string]any); ok {
		variables = copyVariables(locals)
	}
	return ctx.Views.RenderString(lazyview.Options{
		Context:    ctx.Context,
		Request:    ctx.Request,
		Variables:  variables,
		Data:       data,
		Helpers:    ctx.Helpers(),
		Route:      ctx.Route,
		Namespace:  ctx.Namespace,
		Controller: ctx.Controller,
		Partial:    partial,
		Format:     ctx.Format,
		Variants:   ctx.Variants,
		UseLayout:  false,
	})
}

func cacheTurboFrameHelper(ctx *lazyview.Context, args ...any) (any, error) {
	if len(args) < 2 {
		return lazyview.Fragment{}, fmt.Errorf("lazyapp: turbo_frame expects id and data")
	}
	id, ok := args[0].(string)
	if !ok {
		return lazyview.Fragment{}, fmt.Errorf("lazyapp: turbo_frame id must be a string")
	}
	data := args[1]
	var opts []lazyturbo.FrameOption
	var cacheKey string
	for _, arg := range args[2:] {
		switch value := arg.(type) {
		case templateCacheKey:
			cacheKey = string(value)
		case lazyturbo.FrameOption:
			opts = append(opts, value)
		default:
			return lazyview.Fragment{}, fmt.Errorf("lazyapp: unsupported turbo_frame option %T", arg)
		}
	}
	if cacheKey == "" {
		return lazyturbo.Frame(ctx, id, data, opts...)
	}
	parts := cacheContextPrefix(ctx)
	parts = append(parts, cacheKey)
	var err error
	cacheKey, err = lazycache.Key(parts...)
	if err != nil {
		return lazyview.Fragment{}, err
	}
	if err := lazyturbo.ValidateFrameID(id); err != nil {
		return lazyview.Fragment{}, err
	}
	body, err := cachedPartialBody(ctx, cacheKey, strings.TrimSpace(id)+"_frame", data)
	if err != nil {
		return lazyview.Fragment{}, err
	}
	return lazyturbo.FrameTag(id, body, opts...)
}

func copyVariables(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(source))
	maps.Copy(out, source)
	return out
}

func contentTypeForFormat(format string) string {
	switch format {
	case "json":
		return "application/json; charset=utf-8"
	case "svg":
		return "image/svg+xml; charset=utf-8"
	case "turbo_stream":
		return "text/vnd.turbo-stream.html; charset=utf-8"
	default:
		return "text/html; charset=utf-8"
	}
}
