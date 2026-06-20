package lazymailer

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"net/textproto"
	"strings"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazyview"
)

type contextKey struct{}

type Mailer struct {
	views      *lazyview.Views
	deliveries *Registry
}

func New(ctx context.Context, deliveries *Registry) (*Mailer, error) {
	views, ok := lazycontroller.RendererFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("lazymailer: renderer is missing from context")
	}
	return &Mailer{views: views, deliveries: deliveries}, nil
}

func WithContext(ctx context.Context, mailer *Mailer) context.Context {
	return context.WithValue(ctx, contextKey{}, mailer)
}

func FromContext(ctx context.Context) (*Mailer, bool) {
	mailer, ok := ctx.Value(contextKey{}).(*Mailer)
	return mailer, ok
}

type Base struct {
	ctx        context.Context
	mailer     *Mailer
	controller string
	layout     string
	defaults   Defaults
	variables  map[string]any
}

type Defaults struct {
	From     mail.Address
	Delivery string
	Layout   string
	Headers  textproto.MIMEHeader
}

type Options struct {
	Action   string
	To       []mail.Address
	From     mail.Address
	Subject  string
	Delivery string
	Layout   string
	Headers  textproto.MIMEHeader
}

func NewBase(ctx context.Context, controller string, defaults Defaults) (Base, error) {
	mailer, ok := FromContext(ctx)
	if !ok {
		return Base{}, fmt.Errorf("lazymailer: mailer is missing from context")
	}
	if strings.TrimSpace(controller) == "" {
		return Base{}, fmt.Errorf("lazymailer: controller is required")
	}
	return Base{
		ctx:        ctx,
		mailer:     mailer,
		controller: controller,
		layout:     firstNonEmpty(defaults.Layout, "mailer"),
		defaults:   defaults,
		variables:  map[string]any{},
	}, nil
}

func (b *Base) Set(name string, value any) {
	if b.variables == nil {
		b.variables = map[string]any{}
	}
	b.variables[name] = value
}

func (b *Base) Mail(options Options) error {
	message, err := b.Build(options)
	if err != nil {
		return err
	}
	return b.mailer.deliveries.Deliver(b.ctx, firstNonEmpty(options.Delivery, b.defaults.Delivery), message)
}

func (b *Base) Build(options Options) (Message, error) {
	if b.mailer == nil || b.mailer.views == nil {
		return Message{}, fmt.Errorf("lazymailer: base is not initialized")
	}
	action := strings.TrimSpace(options.Action)
	if action == "" {
		return Message{}, fmt.Errorf("lazymailer: action is required")
	}
	layout := firstNonEmpty(options.Layout, b.layout)

	text, err := b.render(action, layout, "text")
	if err != nil && !isMissingView(err) {
		return Message{}, err
	}
	html, err := b.render(action, layout, "html")
	if err != nil && !isMissingView(err) {
		return Message{}, err
	}
	if text == "" && html == "" {
		return Message{}, fmt.Errorf("lazymailer: no templates found for %s.%s", b.controller, action)
	}

	headers := cloneHeader(b.defaults.Headers)
	for key, values := range options.Headers {
		headers.Del(key)
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	return Message{
		From:    firstAddress(options.From, b.defaults.From),
		To:      options.To,
		Subject: options.Subject,
		Headers: headers,
		Text:    text,
		HTML:    html,
	}, nil
}

func (b *Base) render(action string, layout string, format string) (string, error) {
	var out bytes.Buffer
	err := b.mailer.views.Render(lazyview.Options{
		Context:    b.ctx,
		Writer:     &out,
		Variables:  b.variables,
		Controller: b.controller,
		Action:     action,
		Format:     format,
		Layout:     layout,
		UseLayout:  true,
	})
	return out.String(), err
}

func isMissingView(err error) bool {
	return err != nil && strings.Contains(err.Error(), "view not found")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstAddress(values ...mail.Address) mail.Address {
	for _, value := range values {
		if strings.TrimSpace(value.Address) != "" {
			return value
		}
	}
	return mail.Address{}
}
