package harness

import (
	"context"
	"fmt"
)

func RunBoundedAdvisoryLoop(ctx context.Context, maxSteps int, step func(context.Context, int) (bool, error)) error {
	if maxSteps <= 0 {
		return fmt.Errorf("maxSteps must be greater than zero")
	}
	for i := 0; i < maxSteps; i++ {
		done, err := step(ctx, i)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return fmt.Errorf("advisory loop reached max steps (%d)", maxSteps)
}
