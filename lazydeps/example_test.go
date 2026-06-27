package lazydeps_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"golazy.dev/lazydeps"
)

func ExampleScope_Shutdown() {
	deps := lazydeps.New(
		context.Background(),
		lazydeps.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)
	events := []string{}

	database, _ := lazydeps.Service(deps, "database", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		return ctx, "database", nil, func() {
			events = append(events, "database closed: "+context.Cause(ctx).Error())
		}
	})

	_, _ = lazydeps.Service(deps, "posts", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		_ = database.Use()
		return ctx, "posts", nil, func() {
			events = append(events, "posts closed: "+context.Cause(ctx).Error())
		}
	})

	_ = deps.Shutdown(context.Background(), "deploy")
	for _, event := range events {
		fmt.Println(event)
	}

	// Output:
	// posts closed: deploy
	// database closed: deploy
}
