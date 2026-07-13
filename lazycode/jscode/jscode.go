// Package jscode provides deliberately bounded JavaScript text edits. It only
// manages exact single-line imports and explicitly marked GoLazy blocks; it is
// not a general JavaScript parser or rewriter.
package jscode

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golazy.dev/lazycode"
)

const (
	beginPrefix = "// golazy:begin "
	endPrefix   = "// golazy:end "
)

type EditFunc func([]byte) ([]byte, bool, error)

func Edit(name string, edit EditFunc) lazycode.Operation {
	return lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		if edit == nil {
			return errors.New("jscode: edit function is required")
		}
		source, err := workspace.Read(name)
		if err != nil {
			return err
		}
		result, changed, err := edit(source)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return workspace.Replace(name, result)
	})
}

func ManagedBlock(name, id, body string) lazycode.Operation {
	return Edit(name, func(source []byte) ([]byte, bool, error) {
		return EnsureManagedBlock(source, id, body)
	})
}

func Import(name, statement string) lazycode.Operation {
	return Edit(name, func(source []byte) ([]byte, bool, error) {
		return EnsureImport(source, statement)
	})
}

func EnsureManagedBlock(source []byte, id, body string) ([]byte, bool, error) {
	if err := validateID(id); err != nil {
		return nil, false, err
	}
	lines, newline, final := splitLines(source)
	begin, end, err := findBlock(lines, id)
	if err != nil {
		return nil, false, err
	}
	bodyLines := normalizedBody(body)
	beginMarker, endMarker := beginPrefix+id, endPrefix+id
	var result []string
	if begin >= 0 {
		replacement := make([]string, 0, len(bodyLines)+2)
		replacement = append(replacement, beginMarker)
		replacement = append(replacement, bodyLines...)
		replacement = append(replacement, endMarker)
		result = replaceLines(lines, begin, end+1, replacement)
	} else {
		result = append([]string(nil), lines...)
		if len(result) != 0 && strings.TrimSpace(result[len(result)-1]) != "" {
			result = append(result, "")
		}
		result = append(result, beginMarker)
		result = append(result, bodyLines...)
		result = append(result, endMarker)
		final = true
	}
	encoded := joinLines(result, newline, final)
	if string(encoded) == string(source) {
		return append([]byte(nil), source...), false, nil
	}
	return encoded, true, nil
}

func RemoveManagedBlock(source []byte, id string) ([]byte, bool, error) {
	if err := validateID(id); err != nil {
		return nil, false, err
	}
	lines, newline, final := splitLines(source)
	begin, end, err := findBlock(lines, id)
	if err != nil {
		return nil, false, err
	}
	if begin < 0 {
		return append([]byte(nil), source...), false, nil
	}
	if begin > 0 && strings.TrimSpace(lines[begin-1]) == "" {
		begin--
	}
	result := replaceLines(lines, begin, end+1, nil)
	return joinLines(result, newline, final), true, nil
}

// EnsureImport inserts an exact, single-line ES module import. Callers retain
// control over named/default import syntax by supplying the complete statement.
func EnsureImport(source []byte, statement string) ([]byte, bool, error) {
	statement = strings.TrimSpace(statement)
	if err := validateImport(statement); err != nil {
		return nil, false, err
	}
	lines, newline, final := splitLines(source)
	for _, line := range lines {
		if strings.TrimSpace(line) == statement {
			return append([]byte(nil), source...), false, nil
		}
	}
	insertAt := 0
	if len(lines) > 0 && strings.HasPrefix(lines[0], "#!") {
		insertAt = 1
	}
	sawImport := false
	for index := insertAt; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			if !sawImport {
				insertAt = index + 1
			}
			continue
		}
		if strings.HasPrefix(trimmed, "import ") {
			sawImport = true
			insertAt = index + 1
			continue
		}
		break
	}
	lines = insertLines(lines, insertAt, statement)
	if len(lines) > insertAt+1 && strings.TrimSpace(lines[insertAt+1]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[insertAt+1]), "import ") {
		lines = insertLines(lines, insertAt+1, "")
	}
	return joinLines(lines, newline, final), true, nil
}

func EnsureSideEffectImport(source []byte, module string) ([]byte, bool, error) {
	if strings.TrimSpace(module) == "" {
		return nil, false, errors.New("jscode: module is required")
	}
	return EnsureImport(source, "import "+strconv.Quote(module)+";")
}

func RemoveImport(source []byte, statement string) ([]byte, bool, error) {
	statement = strings.TrimSpace(statement)
	if err := validateImport(statement); err != nil {
		return nil, false, err
	}
	lines, newline, final := splitLines(source)
	for index, line := range lines {
		if strings.TrimSpace(line) != statement {
			continue
		}
		lines = replaceLines(lines, index, index+1, nil)
		if index < len(lines) && index > 0 && strings.TrimSpace(lines[index]) == "" && strings.TrimSpace(lines[index-1]) == "" {
			lines = replaceLines(lines, index, index+1, nil)
		}
		return joinLines(lines, newline, final), true, nil
	}
	return append([]byte(nil), source...), false, nil
}

func findBlock(lines []string, id string) (int, int, error) {
	begin, end := -1, -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, beginPrefix) {
			foundID := strings.TrimSpace(strings.TrimPrefix(trimmed, beginPrefix))
			if foundID == id {
				if begin >= 0 {
					return -1, -1, fmt.Errorf("jscode: duplicate managed block %q", id)
				}
				begin = index
			}
		}
		if strings.HasPrefix(trimmed, endPrefix) {
			foundID := strings.TrimSpace(strings.TrimPrefix(trimmed, endPrefix))
			if foundID == id {
				if end >= 0 {
					return -1, -1, fmt.Errorf("jscode: duplicate managed block end %q", id)
				}
				end = index
			}
		}
	}
	if begin < 0 && end < 0 {
		return -1, -1, nil
	}
	if begin < 0 || end < 0 || end < begin {
		return -1, -1, fmt.Errorf("jscode: malformed managed block %q", id)
	}
	return begin, end, nil
}

func validateID(id string) error {
	if id == "" {
		return errors.New("jscode: managed block ID is required")
	}
	for _, character := range id {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || strings.ContainsRune("._/-", character) {
			continue
		}
		return fmt.Errorf("jscode: invalid managed block ID %q", id)
	}
	return nil
}

func validateImport(statement string) error {
	if statement == "" || strings.ContainsAny(statement, "\r\n") || !strings.HasPrefix(statement, "import ") {
		return errors.New("jscode: expected one complete single-line import statement")
	}
	return nil
}

func normalizedBody(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.Trim(body, "\n")
	if body == "" {
		return nil
	}
	return strings.Split(body, "\n")
}

func splitLines(source []byte) ([]string, string, bool) {
	text := string(source)
	newline := "\n"
	if strings.Contains(text, "\r\n") {
		newline = "\r\n"
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	final := strings.HasSuffix(text, "\n")
	if final {
		text = strings.TrimSuffix(text, "\n")
	}
	if text == "" {
		return nil, newline, final
	}
	return strings.Split(text, "\n"), newline, final
}

func joinLines(lines []string, newline string, final bool) []byte {
	text := strings.Join(lines, newline)
	if final && len(lines) != 0 {
		text += newline
	}
	return []byte(text)
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
