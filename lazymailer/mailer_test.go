package lazymailer_test

import (
	"context"
	"testing"
	"testing/fstest"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazymailer"
	_ "golazy.dev/lazyview/gotmpl"
)

func TestMailerRendersViewsAndDelivers(t *testing.T) {
	views, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl":         {Data: []byte(`{{.content}}`)},
		"layouts/mailer.text.tpl":      {Data: []byte(`text layout {{.content}}`)},
		"layouts/mailer.html.tpl":      {Data: []byte(`<html>{{.content}}</html>`)},
		"notice_mailer/hello.text.tpl": {Data: []byte(`Hello {{.name}}`)},
		"notice_mailer/hello.html.tpl": {Data: []byte(`<p>Hello {{.name}}</p>`)},
	})
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	ctx := lazycontroller.WithRenderer(context.Background(), views)
	delivery := &lazymailer.MemoryDelivery{}
	mailer, err := lazymailer.New(ctx, lazymailer.NewRegistry("default", map[string]lazymailer.Delivery{
		"default": delivery,
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx = lazymailer.WithContext(ctx, mailer)

	base, err := lazymailer.NewBase(ctx, "notice_mailer", lazymailer.Defaults{
		From:     lazymailer.MustParseAddress("sender@example.com"),
		Delivery: "default",
		Layout:   "mailer",
	})
	if err != nil {
		t.Fatalf("NewBase: %v", err)
	}
	base.Set("name", "Ada")
	if err := base.Mail(lazymailer.Options{
		Action:  "hello",
		To:      []lazymailer.Address{lazymailer.MustParseAddress("ada@example.com")},
		Subject: "Welcome",
	}); err != nil {
		t.Fatalf("Mail: %v", err)
	}

	messages := delivery.Messages()
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	if got, want := messages[0].Text, "text layout Hello Ada"; got != want {
		t.Fatalf("Text = %q, want %q", got, want)
	}
	if got, want := messages[0].HTML, "<html><p>Hello Ada</p></html>"; got != want {
		t.Fatalf("HTML = %q, want %q", got, want)
	}
}
