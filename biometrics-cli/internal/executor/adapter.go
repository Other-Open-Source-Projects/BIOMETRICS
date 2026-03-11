package executor

import "context"

type Adapter interface {
	Execute(ctx context.Context, runID, agentName, prompt, projectID string) (string, error)
}
