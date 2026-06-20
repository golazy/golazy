package lazyseo

import (
	"fmt"

	"golazy.dev/lazyview"
)

// Helpers returns the template helpers provided by lazyseo.
func Helpers(defaults ...Option) map[string]any {
	defaultMeta := New(defaults...)
	return map[string]any{
		"seo": func(ctx *lazyview.Context) (lazyview.Fragment, error) {
			meta, err := metaFrom(ctx)
			if err != nil {
				return lazyview.Fragment{}, err
			}
			merged := merge(defaultMeta, meta)
			return lazyview.Fragment{
				Body:        render(merged),
				ContentType: htmlContentType,
			}, nil
		},
		"seo_lang": func(ctx *lazyview.Context) (string, error) {
			meta, err := metaFrom(ctx)
			if err != nil {
				return "", err
			}
			merged := merge(defaultMeta, meta)
			return merged.Language, nil
		},
	}
}

func metaFrom(ctx *lazyview.Context) (*Meta, error) {
	if ctx == nil || ctx.Variables == nil {
		return nil, nil
	}
	value, ok := ctx.Variables["seo"]
	if !ok || value == nil {
		return nil, nil
	}
	switch meta := value.(type) {
	case *Meta:
		return meta, nil
	case Meta:
		return &meta, nil
	default:
		return nil, fmt.Errorf("lazyseo: seo variable has type %T, want lazyseo.Meta", value)
	}
}
