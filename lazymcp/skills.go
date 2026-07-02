package lazymcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"path"
	"sort"
	"strings"
)

func (scope *Scope) skillIndexMetadata(ctx context.Context) []any {
	if len(scope.allowedModules(ctx)) == 0 {
		return nil
	}
	return []any{resourceMetadata("skill://index.json", "index", "MCP skills index", "application/json", nil)}
}

func (scope *Scope) readSkillIndex(ctx context.Context) (any, error) {
	var entries []any
	for _, module := range scope.allowedModules(ctx) {
		for skillPath, spec := range module.skills {
			frontmatter, body, err := skillFrontmatter(spec.FS)
			if err != nil {
				return nil, err
			}
			name, _ := frontmatter["name"].(string)
			if name == "" {
				name = path.Base(skillPath)
			}
			uri := "skill://" + module.name + "/" + skillPath + "/SKILL.md"
			sum := sha256.Sum256(body)
			entries = append(entries, map[string]any{
				"url":         uri,
				"digest":      "sha256:" + hex.EncodeToString(sum[:]),
				"frontmatter": frontmatter,
			})
		}
	}
	return resourceReadResult("skill://index.json", "application/json", ResourceContent{Text: minifiedJSON(map[string]any{"skills": entries})}), nil
}

func (scope *Scope) readSkillResource(ctx context.Context, uri string) (any, error) {
	moduleName, skillPath, filePath, err := splitSkillURI(uri)
	if err != nil {
		return nil, err
	}
	module, err := scope.authorizedModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	spec, ok := module.skills[skillPath]
	if !ok {
		return nil, fmt.Errorf("lazymcp: skill %q not found", skillPath)
	}
	data, err := fs.ReadFile(spec.FS, filePath)
	if err != nil {
		return nil, err
	}
	return resourceReadResult(uri, mimeType(filePath), ResourceContent{Text: string(data)}), nil
}

func (scope *Scope) readDirectory(ctx context.Context, uri string) (any, error) {
	moduleName, skillPath, dirPath, err := splitSkillURI(uri)
	if err != nil {
		return nil, err
	}
	module, err := scope.authorizedModule(ctx, moduleName)
	if err != nil {
		return nil, err
	}
	spec, ok := module.skills[skillPath]
	if !ok {
		return nil, fmt.Errorf("lazymcp: skill %q not found", skillPath)
	}
	entries, err := fs.ReadDir(spec.FS, strings.Trim(dirPath, "/"))
	if err != nil {
		return nil, err
	}
	var children []any
	for _, entry := range entries {
		mimeType := "application/octet-stream"
		if entry.IsDir() {
			mimeType = "inode/directory"
		} else {
			mimeType = mimeTypeForName(entry.Name())
		}
		children = append(children, map[string]any{
			"name":     entry.Name(),
			"uri":      strings.TrimRight(uri, "/") + "/" + entry.Name(),
			"mimeType": mimeType,
		})
	}
	return map[string]any{"entries": children}, nil
}

func splitSkillURI(uri string) (string, string, string, error) {
	raw := strings.TrimPrefix(uri, "skill://")
	if raw == uri {
		return "", "", "", fmt.Errorf("lazymcp: invalid skill URI %q", uri)
	}
	parts := strings.Split(raw, "/")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("lazymcp: invalid skill URI %q", uri)
	}
	module := parts[0]
	skill := parts[1]
	file := path.Clean(strings.Join(parts[2:], "/"))
	if file == "." {
		file = ""
	}
	return module, skill, file, nil
}

func skillFrontmatter(fsys fs.FS) (map[string]any, []byte, error) {
	body, err := fs.ReadFile(fsys, "SKILL.md")
	if err != nil {
		return nil, nil, err
	}
	frontmatter := map[string]any{}
	lines := strings.Split(string(body), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return frontmatter, body, nil
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		frontmatter[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return frontmatter, body, nil
}

func mimeType(filePath string) string {
	if path.Base(filePath) == "SKILL.md" {
		return "text/markdown"
	}
	return mimeTypeForName(filePath)
}

func mimeTypeForName(name string) string {
	if typ := mime.TypeByExtension(path.Ext(name)); typ != "" {
		return typ
	}
	switch path.Ext(name) {
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	default:
		return "text/plain"
	}
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func jsonText(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
