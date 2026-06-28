package pg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open creates a pgx pool for databaseURL.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, fmt.Errorf("pg: database URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pg: open database: %w", err)
	}
	return pool, nil
}

// OpenEnv creates a pgx pool from the first non-empty environment variable in
// names. If names is empty, DATABASE_URL is used.
func OpenEnv(ctx context.Context, names ...string) (*pgxpool.Pool, error) {
	if len(names) == 0 {
		names = []string{"DATABASE_URL"}
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return Open(ctx, value)
		}
	}
	return nil, fmt.Errorf("pg: none of the configured database URL variables are set")
}
