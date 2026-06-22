package lazydoc

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

func LoadJSON(root fs.FS, file string) (*Index, error) {
	data, err := fs.ReadFile(root, file)
	if err != nil {
		return nil, fmt.Errorf("read package docs %s: %w", file, err)
	}
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse package docs %s: %w", file, err)
	}
	return &index, nil
}

func LoadJSONBytes(data []byte) (*Index, error) {
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	return &index, nil
}
