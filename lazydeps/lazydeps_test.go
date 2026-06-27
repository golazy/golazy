package lazydeps

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
)

func TestServiceTracksAppAndServiceDependencies(t *testing.T) {
	u := New(context.Background())

	db, err := Service(u, "db", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		return ctx, "db", nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Service(u, "posts", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		return ctx, "posts uses " + db.Use(), nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	graph := u.Graph()
	wantEdges := []Edge{
		{From: "app", To: "db"},
		{From: "app", To: "posts"},
		{From: "posts", To: "db"},
	}
	if !slices.Equal(graph.Edges, wantEdges) {
		t.Fatalf("edges = %#v, want %#v", graph.Edges, wantEdges)
	}
}

func TestServiceCancelsNodeContextOnError(t *testing.T) {
	u := New(context.Background())

	var nodeContext context.Context
	_, err := Service(u, "db", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		nodeContext = ctx
		return ctx, "", errors.New("dial database"), nil
	})
	if err == nil {
		t.Fatal("err = nil, want startup error")
	}
	if nodeContext.Err() == nil {
		t.Fatal("node context was not canceled")
	}
}

func TestRefUsePanicsOutsideServiceInitialization(t *testing.T) {
	u := New(context.Background())

	db, err := Service(u, "db", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		return ctx, "db", nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("Use did not panic")
		}
	}()
	db.Use()
}

func TestShutdownCancelsDependentsBeforeDependencies(t *testing.T) {
	var logs bytes.Buffer
	u := New(context.Background(), WithLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	events := []string{}

	db, err := Service(u, "db", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		return ctx, "db", nil, func() {
			events = append(events, "db:"+context.Cause(ctx).Error())
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Service(u, "posts", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
		_ = db.Use()
		return ctx, "posts", nil, func() {
			events = append(events, "posts:"+context.Cause(ctx).Error())
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := u.Shutdown(context.Background(), "deploy"); err != nil {
		t.Fatal(err)
	}
	want := []string{"posts:deploy", "db:deploy"}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}

	if err := u.Shutdown(context.Background(), "second"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, want) {
		t.Fatalf("events after second shutdown = %#v, want idempotent %#v", events, want)
	}

	output := logs.String()
	for _, part := range []string{
		"lazydeps: canceling service context",
		"service=posts",
		"reason=deploy",
		"lazydeps: service cleanup finished",
		"duration=",
	} {
		if !strings.Contains(output, part) {
			t.Fatalf("logs %q do not contain %q", output, part)
		}
	}
}
