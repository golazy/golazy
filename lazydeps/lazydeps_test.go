package lazydeps

import (
	"context"
	"errors"
	"slices"
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
