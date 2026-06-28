package pgmigrate

import (
	"strings"
	"testing"

	"golazy.dev/lazymigrate"
)

func TestParseLazySections(t *testing.T) {
	sections, err := parse([]byte(`
-- +lazy Up
CREATE TABLE widgets (
	id BIGSERIAL PRIMARY KEY
);

-- +lazy Down
DROP TABLE widgets;
`))
	if err != nil {
		t.Fatal(err)
	}

	up, err := sections.forDirection(lazymigrate.DirectionUp)
	if err != nil {
		t.Fatal(err)
	}
	if up != "CREATE TABLE widgets (\n\tid BIGSERIAL PRIMARY KEY\n);" {
		t.Fatalf("unexpected up section:\n%s", up)
	}

	down, err := sections.forDirection(lazymigrate.DirectionDown)
	if err != nil {
		t.Fatal(err)
	}
	if down != "DROP TABLE widgets;" {
		t.Fatalf("unexpected down section: %q", down)
	}
}

func TestParseRejectsGooseFormat(t *testing.T) {
	_, err := parse([]byte(`
-- +goose Up
CREATE TABLE widgets (id BIGSERIAL PRIMARY KEY);

-- +goose Down
DROP TABLE widgets;
`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "content before the first -- +lazy marker") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRejectsUnknownLazyDirective(t *testing.T) {
	_, err := parse([]byte(`-- +lazy Both`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown lazy migration section") {
		t.Fatalf("unexpected error: %v", err)
	}
}
