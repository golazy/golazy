package lazyfiles

import (
	"context"
	"fmt"
)

// Repository persists file catalog metadata.
type Repository interface {
	Put(context.Context, File, Location, ...any) (File, []any, error)
	Find(context.Context, Query, ...any) (File, []Location, []any, error)
	Delete(context.Context, string, ...any) ([]any, error)
}

func activeLocation(locations []Location) (Location, bool) {
	for _, location := range locations {
		if location.Role == RolePrimary && location.Status == StatusActive {
			return location, true
		}
	}
	for _, location := range locations {
		if location.Status == StatusActive {
			return location, true
		}
	}
	for _, location := range locations {
		if location.Status == "" {
			return location, true
		}
	}
	return Location{}, false
}

func validateFileLocation(file File, location Location) error {
	if file.ID == "" {
		return fmt.Errorf("lazyfiles: file id is required")
	}
	if location.FileID == "" {
		location.FileID = file.ID
	}
	if location.FileID != file.ID {
		return fmt.Errorf("lazyfiles: location file id %q does not match file id %q", location.FileID, file.ID)
	}
	if location.Storage == "" {
		return fmt.Errorf("lazyfiles: storage name is required")
	}
	if location.Key == "" {
		return fmt.Errorf("lazyfiles: storage key is required")
	}
	return nil
}
