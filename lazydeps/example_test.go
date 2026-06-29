package lazydeps_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"golazy.dev/lazydeps"
)

func ExampleService() {
	deps := lazydeps.New(context.Background())

	type database struct {
		Name string
	}

	db, _ := lazydeps.Service(deps, "database", func(ctx context.Context) (context.Context, *database, error, context.CancelFunc) {
		return ctx, &database{Name: "primary"}, nil, nil
	})

	_, _ = lazydeps.Service(deps, "posts", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		database := db.Use()
		fmt.Println("posts uses", database.Name)
		return ctx, "ready", nil, nil
	})

	for _, edge := range deps.Graph().Edges {
		fmt.Println(edge.From, "->", edge.To)
	}

	// Output:
	// posts uses primary
	// app -> database
	// app -> posts
	// posts -> database
}

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
