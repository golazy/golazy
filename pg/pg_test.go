package pg

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPoolContext(t *testing.T) {
	pool := &pgxpool.Pool{}
	ctx := WithPool(context.Background(), pool)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext ok = false, want true")
	}
	if got != pool {
		t.Fatalf("FromContext pool = %p, want %p", got, pool)
	}
}
