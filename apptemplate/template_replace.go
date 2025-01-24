package apptemplate

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (t *Template) Replace(old, news string) {
	t.actions = append(t.actions, &basicAction{
		name: "Replace",
		Fn: func(ctx runCtx) {
			// Replace old with new in all the files in the target directory
			err := filepath.Walk(ctx.Dest, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					content, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					newContent := strings.ReplaceAll(string(content), old, news)
					err = os.WriteFile(path, []byte(newContent), info.Mode())
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Fatalf("error walking the path %q: %v\n", ctx.Dest, err)
			}

		},
	})
}
