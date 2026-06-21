package lazymedia

import (
	"context"
	"fmt"
)

func validateRequest(request Request) error {
	if request.SourceFileID == "" {
		return fmt.Errorf("lazymedia: source file id is required")
	}
	if request.VariantKey == "" {
		return fmt.Errorf("lazymedia: variant key is required")
	}
	return nil
}

func validateVariant(variant Variant) error {
	if variant.SourceFileID == "" {
		return fmt.Errorf("lazymedia: source file id is required")
	}
	if variant.VariantKey == "" {
		return fmt.Errorf("lazymedia: variant key is required")
	}
	return nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
