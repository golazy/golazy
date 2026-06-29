package lazymailer_test

import (
	"context"
	"fmt"
	"testing/fstest"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazymailer"
	_ "golazy.dev/lazyview/gotmpl"
)

func ExampleBase_Mail() {
	renderer, err := lazycontroller.NewRenderer(fstest.MapFS{
		"layouts/app.html.tpl":    {Data: []byte("{{.content}}")},
		"layouts/mailer.html.tpl": {Data: []byte("{{.content}}")},
		"layouts/mailer.text.tpl": {Data: []byte("{{.content}}")},
		"notifier/welcome.html.tpl": {Data: []byte(
			"<p>Hello {{.Name}}</p>",
		)},
		"notifier/welcome.text.tpl": {Data: []byte("Hello {{.Name}}")},
	})
	if err != nil {
		panic(err)
	}

	ctx := lazycontroller.WithRenderer(context.Background(), renderer)
	delivery := &lazymailer.MemoryDelivery{}
	deliveries := lazymailer.NewRegistry("memory", map[string]lazymailer.Delivery{
		"memory": delivery,
	})
	mailer, err := lazymailer.New(ctx, deliveries)
	if err != nil {
		panic(err)
	}
	ctx = lazymailer.WithContext(ctx, mailer)

	base, err := lazymailer.NewBase(ctx, "notifier", lazymailer.Defaults{
		From:   lazymailer.MustParseAddress("GoLazy <hello@example.com>"),
		Layout: "mailer",
	})
	if err != nil {
		panic(err)
	}
	base.Set("Name", "Ada")

	err = base.Mail(lazymailer.Options{
		Action:  "welcome",
		To:      []lazymailer.Address{lazymailer.MustParseAddress("Ada <ada@example.com>")},
		Subject: "Welcome",
	})
	if err != nil {
		panic(err)
	}

	message := delivery.Messages()[0]
	fmt.Println(message.Text)
	fmt.Println(message.HTML)
	// Output:
	// Hello Ada
	// <p>Hello Ada</p>
}
