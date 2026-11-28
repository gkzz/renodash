package command

import "context"

// githubClient provides the GitHub operations used by the get command.
// Callers own request deadlines and response validation.
type githubClient interface {
	FetchIssue(context.Context, string, int) ([]byte, error)
	FetchPR(context.Context, string, int) ([]byte, error)
}
