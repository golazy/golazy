// Package lazyview provides a simple view rendering engine.
package lazyview

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type Views struct {
	FS      fs.FS
	Engines map[string]Engine

	Helpers     map[string]any
	SearchPaths []string
}

func (v *Views) RenderTemplate(ctx context.Context, w io.Writer, vars map[string]any, file string) error {
	ext := strings.TrimPrefix(filepath.Ext(file), ".")
	engine, ok := v.Engines[ext]
	if !ok {
		return fmt.Errorf("lazyview: no engine for %q", ext)
	}
	err := engine.Render(ctx, v, w, vars, v.Helpers, file)
	return err
}

const bufferSize = 1024

// Create a pool for buffers
var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, bufferSize))
	},
}

func (v *Views) Render(opts Options) error {
	var layout string
	var originalW io.Writer
	var buffer *bytes.Buffer
	var file, format string
	var err error
	if len(opts.Content) > 0 {
		if !opts.UseLayout {
			_, err := opts.Writer.Write([]byte(opts.Content))
			return err
		}
		goto RenderLayout
	}

	file, format, err = v.findTemplate(opts)
	if err != nil {
		return err
	}
	if !opts.UseLayout {
		return v.RenderTemplate(opts.Ctx, opts.Writer, opts.Variables, file)
	}
	buffer = bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)
	originalW = opts.Writer

	err = v.RenderTemplate(opts.Ctx, buffer, opts.Variables, file)
	if err != nil {
		return err
	}

	opts.Formats = []string{format}
	opts.Writer = originalW
	layout, err = v.findLayout(opts)
	if err != nil {
		return err
	}

	opts.Content = buffer.String()

RenderLayout:

	//TODO: the template.HTML is only valid for tpl templates
	if opts.Variables == nil {
		opts.Variables = map[string]any{}
	}
	opts.Variables["Content"] = template.HTML(opts.Content)

	return v.RenderTemplate(opts.Ctx, originalW, opts.Variables, layout)
}

func (v *Views) findLayout(opts Options) (string, error) {
	// layouts/namespace/layout.format(+variant).engine
	// layouts/layout.format(+variant).engine
	// layouts/namespace/controller.format(+variant).engine
	// layouts/controller.format(+variant).engine
	// layouts/application.html(+variant).tpl

	layouts := []string{}
	if opts.Layout != "" {
		layouts = append(layouts, opts.Layout)
	} else {
		if opts.Controller != "" {
			layouts = append(layouts, opts.Controller)
		}
		layouts = append(layouts, "application")
	}

	variants := opts.Variants
	if len(variants) == 0 {
		variants = []string{""}
	}

	namespaces := []string{}
	if opts.Namespace != "" {
		namespaces = append(namespaces, opts.Namespace)
	}
	namespaces = append(namespaces, "")

	sp := v.SearchPaths
	if len(sp) == 0 {
		sp = []string{""}
	}
	tries := []string{}
	for _, layout := range layouts {
		for _, dir := range namespaces {
			for _, variant := range variants {
				for _, spath := range sp {

					filename := layout
					if len(opts.Formats) > 0 {
						filename += "." + opts.Formats[0]
					}
					if variant != "" {
						filename += "+" + variant
					}
					pattern := path.Join(spath, "layouts", dir, filename+".*")
					tries = append(tries, pattern)
					files, err := fs.Glob(v.FS, pattern)
					if err != nil {
						return "", err
					}
					for _, file := range files {
						ext := strings.TrimPrefix(filepath.Ext(file), ".")
						_, ok := v.Engines[ext]
						if !ok {
							continue
						}
						return file, nil
					}

				}

			}

		}
	}

	return "", fmt.Errorf("lazyview: layout not found. Tried: %s", strings.Join(tries, ", "))
}
func (v *Views) findTemplate(opts Options) (file, format string, err error) {

	// Set formats based on mime types
	if len(opts.Formats) == 0 {
		if len(opts.Accept) == 0 || opts.Accept == "*/*" {
			opts.Formats = []string{"html", "json"}
		} else {
			opts.Formats = getExtensionsForMIMETypes(opts.Accept)
		}
	}

	name := opts.Action
	if opts.Action == "" {
		if opts.Partial == "" {
			return "", "", fmt.Errorf("lazyview: no action or partial defined")
		}
		name = "_" + opts.Partial
	}

	// For each search path
	// namespace/controller/(action|_partial).format(+variant).engine
	// namespace/(action|_partial).format(+variant).engine
	// "application"/(action|_partial).format(+variant).engine

	variants := opts.Variants
	if len(variants) == 0 {
		variants = []string{""}
	}

	sp := v.SearchPaths
	if len(sp) == 0 {
		sp = []string{""}
	}

	tries := []string{}
	// For each format
	for _, format := range opts.Formats {

		for _, dir := range []string{
			path.Join(opts.Namespace, opts.Controller),
			path.Join(opts.Namespace),
			path.Join("application"),
		} {
			if dir == "" {
				continue
			}
			// For each variant
			for _, variant := range variants {

				// For each search path
				for _, spath := range sp {

					filename := name
					if format != "" {
						filename += "." + format
					}
					if variant != "" {
						filename += "+" + variant
					}

					pattern := path.Join(spath, dir, filename+".*")
					tries = append(tries, pattern)
					files, err := fs.Glob(v.FS, pattern)
					if err != nil {
						return "", "", err
					}
					for _, file := range files {

						ext := strings.TrimPrefix(filepath.Ext(file), ".")
						_, ok := v.Engines[ext]
						if !ok {
							continue
						}
						return file, format, nil
					}
				}
			}
		}
	}
	return "", "", fmt.Errorf("lazyview: template not found. Tried %s", strings.Join(tries, ", "))
}

func getExtensionsForMIMETypes(acceptHeader string) []string {
	var extensions []string

	// Split the Accept header into MIME types
	mimeTypes := strings.Split(acceptHeader, ",")

	for _, mimeType := range mimeTypes {
		// Remove any quality parameters
		mimeType = strings.Split(mimeType, ";")[0]
		// Get the file extension for the MIME type
		exts, err := mime.ExtensionsByType(strings.TrimSpace(mimeType))
		if err != nil {
			//fmt.Println("Error finding extensions for MIME type:", mimeType, err)
			continue
		}
		for i, ext := range exts {
			exts[i] = strings.TrimPrefix(ext, ".")
		}
		extensions = append(extensions, exts...)
	}

	return extensions
}

// Options is the set of options to render a view.
// In case there are several views that match the search criteria, this is the list of priorities:
// SearchPaths > Variants > Namespace/Controller|Namespace|application > Formats
type Options struct {
	Ctx    context.Context
	Writer io.Writer

	Variables map[string]any

	// Content represents the view content. If defined, Action and Partial are ignored as only the layout is rendred
	Content string

	// Action to render
	Action string

	// Partial to render. If Action is defined, this is ignored.
	Partial string

	Controller string
	Namespace  string

	// Variants allows to specified a variant preference for the view.
	// For example, a mobile variant will expect the view to be named as "index.html+mobile.tpl"
	// If empty, no variants are used.
	// To allow the default variant, use an empty string.
	Variants []string

	// Formats is the list of formats to search for. Eachone is represented as a file extension.
	// If empty, and Accept is not defined, it will default to ["html", "json"]
	Formats []string
	// Acept is the Accept header value. It is only read is Format is empty.
	// Each mime type is converted to the common file extension as defined by the mime package.
	Accept string

	// UseLayout determines if the layout should be used.
	UseLayout bool

	// Layout defines which layout should be used.
	// If empty the default layout will be used.
	Layout string
}
