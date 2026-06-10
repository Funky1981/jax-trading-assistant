package harness

import "context"

type StepDecision struct {
	NeedTool bool
	ToolName string
}

func RunBoundedAdvisoryLoop(ctx context.Context, maxSteps int, step func(context.Context, int) (bool, error)) error {
	for i := 0; i < maxSteps; i++ {
		done, err := step(ctx, i)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return nil
}
