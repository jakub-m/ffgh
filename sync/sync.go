package sync

import (
	"encoding/json"
	"ffgh/config"
	"ffgh/gh"
	"ffgh/storage"
	"fmt"
	"log"
	"os/exec"
	"time"
)

type Synchronizer struct {
	Storage  storage.Storage
	Interval time.Duration
}

const jsonFields = "author,body,commentsCount,createdAt,id,number,repository,state,title,updatedAt,url"

// assignees
// author
// authorAssociation
// body
// closedAt
// commentsCount
// createdAt
// id
// isDraft
// isLocked
// isPullRequest
// labels
// number
// repository
// state
// title
// updatedAt
// url

func New() *Synchronizer {
	return &Synchronizer{
		Storage:  storage.NewFileStorage(),
		Interval: 60 * time.Second,
	}
}

func (s *Synchronizer) RunBlocking(config config.Config) error {
	for {
		if err := s.RunOnce(config); err != nil {
			return err
		}
		time.Sleep(s.Interval)
	}
}

func (s *Synchronizer) RunOnce(config config.Config) error {
	log.Printf("Run gh search")
	allPrs := []gh.PullRequest{}
	urlsVisited := make(map[string]bool)
	for _, q := range config.Queries {
		prs, err := getPrs(q.GitHubArg, q.QueryName)
		if err != nil {
			return fmt.Errorf("error while querying PRs %s: %w", q.GitHubArg, err)
		}
		for _, pr := range prs {
			if urlsVisited[pr.URL] {
				continue
			}
			urlsVisited[pr.URL] = true
			allPrs = append(allPrs, pr)
		}
	}
	if err := s.Storage.ResetPullRequests(allPrs); err != nil {
		return fmt.Errorf("error while storing PRs: %w", err)
	}

	log.Printf("Updated %d pull requests", len(allPrs))
	return nil
}

func getPrs(query, metaLabel string) ([]gh.PullRequest, error) {
	log.Printf("Get PRs for: %s", query)
	out, err := exec.Command("gh", "search", "prs", "--state=open", query, "--json", jsonFields).Output()
	if err != nil {
		return nil, fmt.Errorf("error while running gh command: %s", err)
	}
	var prs []gh.PullRequest
	err = json.Unmarshal(out, &prs)
	if err != nil {
		return nil, fmt.Errorf("error while interpreting JSON output of gh command: %s\n\n%s", err, out)
	}
	for i, pr := range prs {
		pr.Meta.Label = metaLabel
		prs[i] = pr
	}
	return prs, nil
}
