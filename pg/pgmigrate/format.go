package pgmigrate

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"golazy.dev/lazymigrate"
)

const (
	lazyDirective = "+lazy"
)

type sections struct {
	up   string
	down string
}

func parse(content []byte) (sections, error) {
	var parsed sections
	var current lazymigrate.Direction
	var sawMarker bool
	var up strings.Builder
	var down strings.Builder

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	sawUp := false
	sawDown := false
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		if direction, ok, err := parseMarker(line); err != nil {
			return sections{}, fmt.Errorf("pgmigrate: line %d: %w", lineNumber, err)
		} else if ok {
			current = direction
			sawMarker = true
			if direction == lazymigrate.DirectionUp {
				sawUp = true
			}
			if direction == lazymigrate.DirectionDown {
				sawDown = true
			}
			continue
		}

		switch current {
		case lazymigrate.DirectionUp:
			up.WriteString(line)
			up.WriteByte('\n')
		case lazymigrate.DirectionDown:
			down.WriteString(line)
			down.WriteByte('\n')
		default:
			if strings.TrimSpace(line) != "" {
				return sections{}, fmt.Errorf("pgmigrate: content before the first -- +lazy marker is not allowed")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return sections{}, fmt.Errorf("pgmigrate: read migration: %w", err)
	}
	if !sawMarker {
		return sections{}, fmt.Errorf("pgmigrate: migration must contain -- +lazy Up and -- +lazy Down markers")
	}
	if !sawUp || !sawDown {
		return sections{}, fmt.Errorf("pgmigrate: migration must contain both -- +lazy Up and -- +lazy Down markers")
	}

	parsed.up = strings.TrimSpace(up.String())
	parsed.down = strings.TrimSpace(down.String())
	return parsed, nil
}

func (parsed sections) forDirection(direction lazymigrate.Direction) (string, error) {
	switch direction {
	case lazymigrate.DirectionUp:
		if parsed.up == "" {
			return "", fmt.Errorf("pgmigrate: -- +lazy Up section is empty")
		}
		return parsed.up, nil
	case lazymigrate.DirectionDown:
		if parsed.down == "" {
			return "", fmt.Errorf("pgmigrate: -- +lazy Down section is empty")
		}
		return parsed.down, nil
	default:
		return "", fmt.Errorf("pgmigrate: unsupported direction %q", direction)
	}
}

func parseMarker(line string) (lazymigrate.Direction, bool, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "--") {
		return "", false, nil
	}
	directive := strings.TrimSpace(strings.TrimPrefix(line, "--"))
	if !strings.HasPrefix(directive, lazyDirective) {
		return "", false, nil
	}
	fields := strings.Fields(directive)
	if len(fields) != 2 || fields[0] != lazyDirective {
		return "", false, fmt.Errorf("invalid lazy marker %q", line)
	}
	switch fields[1] {
	case "Up":
		return lazymigrate.DirectionUp, true, nil
	case "Down":
		return lazymigrate.DirectionDown, true, nil
	default:
		return "", false, fmt.Errorf("unknown lazy migration section %q", fields[1])
	}
}
