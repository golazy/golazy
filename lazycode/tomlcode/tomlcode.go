package tomlcode

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golazy.dev/lazycode"
)

// Document is a parsed TOML document that retains its original line endings,
// comments, and unrelated source text.
type Document struct {
	lines        []string
	newline      string
	finalNewline bool
}

type section struct {
	name  string
	start int
	end   int
	array bool
}

type entry struct {
	table string
	key   string
	start int
	end   int
}

type documentIndex struct {
	sections []section
	entries  []entry
}

// EditFunc mutates a Document and reports whether it changed.
type EditFunc func(*Document) (bool, error)

// Parse validates data as a document supported by this conservative editor.
func Parse(data []byte) (*Document, error) {
	text := string(data)
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	final := strings.HasSuffix(text, "\n")
	if final {
		text = strings.TrimSuffix(text, "\n")
	}
	var lines []string
	if text != "" {
		lines = strings.Split(text, "\n")
	}
	document := &Document{lines: lines, newline: newline, finalNewline: final}
	if _, err := document.index(); err != nil {
		return nil, err
	}
	return document, nil
}

// Bytes returns the document's current source representation.
func (d *Document) Bytes() []byte {
	if d == nil {
		return nil
	}
	text := strings.Join(d.lines, d.newline)
	if d.finalNewline && (len(d.lines) != 0 || text != "") {
		text += d.newline
	}
	return []byte(text)
}

// EnsureTable creates an ordinary table when it is absent and reports whether
// the document changed.
func (d *Document) EnsureTable(table string) (bool, error) {
	if d == nil {
		return false, errors.New("tomlcode: nil document")
	}
	if err := validateName(table, "table"); err != nil {
		return false, err
	}
	index, err := d.index()
	if err != nil {
		return false, err
	}
	for _, candidate := range index.sections {
		if candidate.name == table {
			if candidate.array {
				return false, fmt.Errorf("tomlcode: %q is an array table", table)
			}
			return false, nil
		}
	}
	if len(d.lines) != 0 && strings.TrimSpace(d.lines[len(d.lines)-1]) != "" {
		d.lines = append(d.lines, "")
	}
	d.lines = append(d.lines, "["+table+"]")
	d.finalNewline = true
	return true, nil
}

// HasTable reports whether an ordinary table exists. The root table always
// exists. Array tables do not satisfy an ordinary-table lookup.
func (d *Document) HasTable(table string) (bool, error) {
	if d == nil {
		return false, errors.New("tomlcode: nil document")
	}
	if table == "" {
		return true, nil
	}
	if err := validateName(table, "table"); err != nil {
		return false, err
	}
	index, err := d.index()
	if err != nil {
		return false, err
	}
	_, ok := findSection(index.sections, table)
	return ok, nil
}

// SetRaw sets a single-line TOML value, creating an ordinary table if needed.
func (d *Document) SetRaw(table, key, value string) (bool, error) {
	if d == nil {
		return false, errors.New("tomlcode: nil document")
	}
	if table != "" {
		if err := validateName(table, "table"); err != nil {
			return false, err
		}
	}
	if err := validateName(key, "key"); err != nil {
		return false, err
	}
	if err := validateValue(value); err != nil {
		return false, err
	}
	changed := false
	if table != "" {
		ensured, err := d.EnsureTable(table)
		if err != nil {
			return false, err
		}
		changed = ensured
	}
	index, err := d.index()
	if err != nil {
		return false, err
	}
	for _, candidate := range index.entries {
		if candidate.table != table || candidate.key != key {
			continue
		}
		line := d.lines[candidate.start]
		equals := findEquals(line)
		if equals < 0 {
			return false, fmt.Errorf("tomlcode: malformed assignment for %s.%s", table, key)
		}
		rest := line[equals+1:]
		leadingLength := len(rest) - len(strings.TrimLeftFunc(rest, unicode.IsSpace))
		leading := rest[:leadingLength]
		if leading == "" {
			leading = " "
		}
		suffix := ""
		if comment := inlineComment(rest); comment >= 0 {
			start := comment
			for start > 0 && (rest[start-1] == ' ' || rest[start-1] == '\t') {
				start--
			}
			suffix = rest[start:]
		}
		replacement := line[:equals+1] + leading + value + suffix
		if candidate.start == candidate.end && replacement == line {
			return changed, nil
		}
		d.lines = replaceLines(d.lines, candidate.start, candidate.end+1, []string{replacement})
		return true, nil
	}

	section, ok := findSection(index.sections, table)
	if !ok {
		return false, fmt.Errorf("tomlcode: table %q not found", table)
	}
	insertAt := section.end
	for insertAt > section.start+1 && strings.TrimSpace(d.lines[insertAt-1]) == "" {
		insertAt--
	}
	d.lines = insertLines(d.lines, insertAt, key+" = "+value)
	d.finalNewline = true
	return true, nil
}

// SetString sets table.key to a TOML string.
func (d *Document) SetString(table, key, value string) (bool, error) {
	return d.SetRaw(table, key, EncodeString(value))
}

// SetBool sets table.key to a TOML boolean.
func (d *Document) SetBool(table, key string, value bool) (bool, error) {
	return d.SetRaw(table, key, strconv.FormatBool(value))
}

// SetInteger sets table.key to a TOML integer.
func (d *Document) SetInteger(table, key string, value int64) (bool, error) {
	return d.SetRaw(table, key, strconv.FormatInt(value, 10))
}

// SetStrings sets table.key to a single-line array of TOML strings.
func (d *Document) SetStrings(table, key string, values []string) (bool, error) {
	return d.SetRaw(table, key, EncodeStrings(values))
}

// Raw returns the source value assigned to table.key in a normalized
// single-line form. It is intended for ownership-aware planners that must
// restore an application's previous value without interpreting arbitrary TOML
// types. Comments inside a multi-line value are omitted from the normalized
// result while the value itself remains valid TOML.
func (d *Document) Raw(table, key string) (string, bool, error) {
	if d == nil {
		return "", false, errors.New("tomlcode: nil document")
	}
	index, err := d.index()
	if err != nil {
		return "", false, err
	}
	for _, candidate := range index.entries {
		if candidate.table != table || candidate.key != key {
			continue
		}
		first := d.lines[candidate.start]
		equals := findEquals(first)
		if equals < 0 {
			return "", false, fmt.Errorf("tomlcode: malformed assignment for %s.%s", table, key)
		}
		parts := make([]string, 0, candidate.end-candidate.start+1)
		for line := candidate.start; line <= candidate.end; line++ {
			value := d.lines[line]
			if line == candidate.start {
				value = value[equals+1:]
			}
			if comment := inlineComment(value); comment >= 0 {
				value = value[:comment]
			}
			if value = strings.TrimSpace(value); value != "" {
				parts = append(parts, value)
			}
		}
		value := strings.Join(parts, " ")
		if err := validateValue(value); err != nil {
			return "", false, fmt.Errorf("tomlcode: read %s.%s: %w", table, key, err)
		}
		return value, true, nil
	}
	return "", false, nil
}

// Remove removes table.key and reports whether it existed.
func (d *Document) Remove(table, key string) (bool, error) {
	if d == nil {
		return false, errors.New("tomlcode: nil document")
	}
	index, err := d.index()
	if err != nil {
		return false, err
	}
	for _, candidate := range index.entries {
		if candidate.table == table && candidate.key == key {
			d.lines = replaceLines(d.lines, candidate.start, candidate.end+1, nil)
			return true, nil
		}
	}
	return false, nil
}

// RemoveTable removes an ordinary table and all of its keys.
func (d *Document) RemoveTable(table string) (bool, error) {
	if d == nil {
		return false, errors.New("tomlcode: nil document")
	}
	if table == "" {
		return false, errors.New("tomlcode: cannot remove the root table")
	}
	index, err := d.index()
	if err != nil {
		return false, err
	}
	section, ok := findSection(index.sections, table)
	if !ok {
		return false, nil
	}
	start, end := section.start, section.end
	if start > 0 && strings.TrimSpace(d.lines[start-1]) == "" {
		start--
	}
	d.lines = replaceLines(d.lines, start, end, nil)
	return true, nil
}

// RemoveTableIfEmpty removes an ordinary table only when it contains no keys.
// It is useful when an ownership-aware edit created a table but must preserve
// application keys added there later.
func (d *Document) RemoveTableIfEmpty(table string) (bool, error) {
	if d == nil {
		return false, errors.New("tomlcode: nil document")
	}
	if table == "" {
		return false, nil
	}
	index, err := d.index()
	if err != nil {
		return false, err
	}
	if _, ok := findSection(index.sections, table); !ok {
		return false, nil
	}
	for _, candidate := range index.entries {
		if candidate.table == table {
			return false, nil
		}
	}
	return d.RemoveTable(table)
}

// Edit returns an operation that parses name, applies edit, validates the
// result, and replaces the file in memory only when it changed.
func Edit(name string, edit EditFunc) lazycode.Operation {
	return lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		if edit == nil {
			return errors.New("tomlcode: edit function is required")
		}
		source, err := workspace.Read(name)
		if err != nil {
			return err
		}
		document, err := Parse(source)
		if err != nil {
			return fmt.Errorf("tomlcode: parse %s: %w", name, err)
		}
		changed, err := edit(document)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		if _, err := document.index(); err != nil {
			return fmt.Errorf("tomlcode: validate %s: %w", name, err)
		}
		return workspace.Replace(name, document.Bytes())
	})
}

// Set returns an operation that assigns a validated raw TOML value.
func Set(name, table, key, value string) lazycode.Operation {
	return Edit(name, func(document *Document) (bool, error) {
		return document.SetRaw(table, key, value)
	})
}

// SetString returns an operation that sets one TOML string value.
func SetString(name, table, key, value string) lazycode.Operation {
	return Edit(name, func(document *Document) (bool, error) {
		return document.SetString(table, key, value)
	})
}

// SetStrings returns an operation that sets one TOML string-array value.
func SetStrings(name, table, key string, values []string) lazycode.Operation {
	values = append([]string(nil), values...)
	return Edit(name, func(document *Document) (bool, error) {
		return document.SetStrings(table, key, values)
	})
}

// Remove returns an operation that removes table.key from name.
func Remove(name, table, key string) lazycode.Operation {
	return Edit(name, func(document *Document) (bool, error) {
		return document.Remove(table, key)
	})
}

func (d *Document) index() (documentIndex, error) {
	root := section{name: "", start: -1, end: len(d.lines)}
	result := documentIndex{sections: []section{root}}
	current := ""
	currentScope := -1
	ordinaryTables := map[string]bool{"": true}
	keys := make(map[string]bool)

	for line := 0; line < len(d.lines); line++ {
		trimmed := strings.TrimSpace(d.lines[line])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if name, array, ok, err := parseHeader(trimmed); ok || err != nil {
			if err != nil {
				return documentIndex{}, fmt.Errorf("line %d: %w", line+1, err)
			}
			if !array && ordinaryTables[name] {
				return documentIndex{}, fmt.Errorf("line %d: duplicate table %q", line+1, name)
			}
			if !array {
				ordinaryTables[name] = true
			}
			result.sections[len(result.sections)-1].end = line
			result.sections = append(result.sections, section{name: name, start: line, end: len(d.lines), array: array})
			current = name
			currentScope = line
			continue
		}
		equals := findEquals(d.lines[line])
		if equals < 0 {
			return documentIndex{}, fmt.Errorf("line %d: unsupported or malformed TOML", line+1)
		}
		key := strings.TrimSpace(d.lines[line][:equals])
		if err := validateName(key, "key"); err != nil {
			return documentIndex{}, fmt.Errorf("line %d: %w", line+1, err)
		}
		end, err := valueEnd(d.lines, line, equals+1)
		if err != nil {
			return documentIndex{}, fmt.Errorf("line %d: %w", line+1, err)
		}
		scopeKey := fmt.Sprintf("%d\x00%s", currentScope, key)
		if keys[scopeKey] {
			return documentIndex{}, fmt.Errorf("line %d: duplicate key %q in table %q", line+1, key, current)
		}
		keys[scopeKey] = true
		result.entries = append(result.entries, entry{table: current, key: key, start: line, end: end})
		line = end
	}
	return result, nil
}

func parseHeader(line string) (name string, array bool, ok bool, err error) {
	withoutComment := line
	if index := inlineComment(line); index >= 0 {
		withoutComment = strings.TrimSpace(line[:index])
	}
	if !strings.HasPrefix(withoutComment, "[") {
		return "", false, false, nil
	}
	if strings.HasPrefix(withoutComment, "[[") {
		if !strings.HasSuffix(withoutComment, "]]") {
			return "", true, true, errors.New("malformed array table")
		}
		name = strings.TrimSpace(withoutComment[2 : len(withoutComment)-2])
		array = true
	} else {
		if !strings.HasSuffix(withoutComment, "]") {
			return "", false, true, errors.New("malformed table")
		}
		name = strings.TrimSpace(withoutComment[1 : len(withoutComment)-1])
	}
	if err := validateName(name, "table"); err != nil {
		return "", array, true, err
	}
	return name, array, true, nil
}

func findEquals(line string) int {
	quote := byte(0)
	escaped := false
	for index := 0; index < len(line); index++ {
		character := line[index]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if quote == '"' && character == '\\' {
				escaped = true
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '"' || character == '\'' {
			quote = character
			continue
		}
		if character == '=' {
			return index
		}
		if character == '#' {
			return -1
		}
	}
	return -1
}

func valueEnd(lines []string, startLine, startColumn int) (int, error) {
	depthSquare, depthBrace := 0, 0
	quote := byte(0)
	triple := false
	escaped := false
	seen := false
	for line := startLine; line < len(lines); line++ {
		column := 0
		if line == startLine {
			column = startColumn
		}
		for column < len(lines[line]) {
			character := lines[line][column]
			if quote != 0 {
				if triple && column+2 < len(lines[line]) && lines[line][column] == quote && lines[line][column+1] == quote && lines[line][column+2] == quote {
					quote, triple = 0, false
					column += 3
					continue
				}
				if !triple && escaped {
					escaped = false
					column++
					continue
				}
				if !triple && quote == '"' && character == '\\' {
					escaped = true
					column++
					continue
				}
				if !triple && character == quote {
					quote = 0
				}
				column++
				continue
			}
			if character == '#' {
				break
			}
			if unicode.IsSpace(rune(character)) {
				column++
				continue
			}
			seen = true
			if (character == '"' || character == '\'') && column+2 < len(lines[line]) && lines[line][column+1] == character && lines[line][column+2] == character {
				quote, triple = character, true
				column += 3
				continue
			}
			if character == '"' || character == '\'' {
				quote = character
				column++
				continue
			}
			switch character {
			case '[':
				depthSquare++
			case ']':
				depthSquare--
			case '{':
				depthBrace++
			case '}':
				depthBrace--
			}
			if depthSquare < 0 || depthBrace < 0 {
				return 0, errors.New("unbalanced TOML value")
			}
			column++
		}
		if quote != 0 && !triple {
			return 0, errors.New("unterminated quoted value")
		}
		if quote == 0 && depthSquare == 0 && depthBrace == 0 {
			if !seen {
				return 0, errors.New("value is required")
			}
			return line, nil
		}
	}
	return 0, errors.New("unterminated TOML value")
}

func inlineComment(text string) int {
	quote := byte(0)
	escaped := false
	for index := 0; index < len(text); index++ {
		character := text[index]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if quote == '"' && character == '\\' {
				escaped = true
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '"' || character == '\'' {
			quote = character
			continue
		}
		if character == '#' {
			return index
		}
	}
	return -1
}

func validateName(name, kind string) error {
	if !validDottedName(name) {
		return fmt.Errorf("tomlcode: invalid %s %q", kind, name)
	}
	return nil
}

func validDottedName(name string) bool {
	if name == "" || strings.TrimSpace(name) != name {
		return false
	}
	for index := 0; index < len(name); {
		if name[index] == '"' || name[index] == '\'' {
			quote := name[index]
			index++
			closed := false
			for index < len(name) {
				if quote == '"' && name[index] == '\\' {
					index += 2
					continue
				}
				if index < len(name) && name[index] == quote {
					index++
					closed = true
					break
				}
				index++
			}
			if !closed {
				return false
			}
		} else {
			start := index
			for index < len(name) && (name[index] == '_' || name[index] == '-' || name[index] >= 'a' && name[index] <= 'z' || name[index] >= 'A' && name[index] <= 'Z' || name[index] >= '0' && name[index] <= '9') {
				index++
			}
			if index == start {
				return false
			}
		}
		if index == len(name) {
			return true
		}
		if name[index] != '.' {
			return false
		}
		index++
		if index == len(name) {
			return false
		}
	}
	return false
}

func validateValue(value string) error {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") {
		return errors.New("tomlcode: value must be a non-empty single line")
	}
	_, err := valueEnd([]string{"key = " + value}, 0, len("key = "))
	if err != nil {
		return fmt.Errorf("tomlcode: invalid value: %w", err)
	}
	return nil
}

// EncodeString renders value as a TOML basic string.
func EncodeString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, character := range value {
		switch character {
		case '\b':
			builder.WriteString(`\b`)
		case '\t':
			builder.WriteString(`\t`)
		case '\n':
			builder.WriteString(`\n`)
		case '\f':
			builder.WriteString(`\f`)
		case '\r':
			builder.WriteString(`\r`)
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		default:
			switch {
			case character < 0x20 || character == 0x7f:
				fmt.Fprintf(&builder, `\u%04X`, character)
			case character > 0xffff:
				fmt.Fprintf(&builder, `\U%08X`, character)
			default:
				builder.WriteRune(character)
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

// EncodeStrings renders values as a single-line TOML string array.
func EncodeStrings(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = EncodeString(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func findSection(sections []section, name string) (section, bool) {
	for _, candidate := range sections {
		if candidate.name == name && !candidate.array {
			return candidate, true
		}
	}
	return section{}, false
}

func insertLines(lines []string, at int, values ...string) []string {
	result := make([]string, 0, len(lines)+len(values))
	result = append(result, lines[:at]...)
	result = append(result, values...)
	result = append(result, lines[at:]...)
	return result
}

func replaceLines(lines []string, start, end int, values []string) []string {
	result := make([]string, 0, len(lines)-(end-start)+len(values))
	result = append(result, lines[:start]...)
	result = append(result, values...)
	result = append(result, lines[end:]...)
	return result
}
