package storage

import (
	"ffgh/gh"
	"time"
)

type Storage interface {
	// ResetPullRequests purges the storage and sets the new pull request.
	ResetPullRequests(prs []gh.PullRequest) error
	GetPullRequests() ([]gh.PullRequest, error)
	MarkUrlAsOpened(url string) error
	MarkUrlAsMuted(url string) error
	GetUserPrState() (*UserPrState, error)
	// GetSyncTime returns last time the state was synchronised and ok (bool) if it was synchronised at all.
	GetSyncTime() (time.Time, bool)
	AddNote(url, note string) error
}
