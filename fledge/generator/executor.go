package generator

import (
	"context"
	"fmt"
	"io"
	"os"
)

// ExecuteOptions configures execution behavior
type ExecuteOptions struct {
	DryRun bool
	Force  bool
	Writer io.Writer // Where to write output (defaults to os.Stdout)
}

// Execute runs operations with validation
func Execute(ctx context.Context, ops []Operation, opts ExecuteOptions) error {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	// Phase 1: Validate all operations
	for _, op := range ops {
		if err := op.Validate(ctx, opts.Force); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Phase 2: Execute or report
	for _, op := range ops {
		if opts.DryRun {
			fmt.Fprintf(opts.Writer, "✓ [DRY RUN] %s\n", op.Description())
		} else {
			if err := op.Execute(ctx); err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}
			fmt.Fprintf(opts.Writer, "✓ %s\n", op.Description())
		}
	}

	return nil
}
