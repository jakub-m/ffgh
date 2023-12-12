package sync

import (
	"encoding/json"
	"ffgh/config"
	"ffgh/gh"
	"ffgh/storage"
	"fmt"
	"log"
	"os/exec"
	"strings"
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

// RunOnce synchronizes state of the GH PRs once. The same PR (same URL) can appear in many queries. The method
// returns only a single PR and uses the attribution order from config to figure which query should it attributre
// the PR to.
func (s *Synchronizer) RunOnce(config config.Config) error {
	log.Printf("Run gh search")
	// All the PRs with duplicates from different queries.
	queriedPrs := make(map[string][]gh.PullRequest)
	for _, q := range config.Queries {
		prs, err := getPrs(q.GitHubArg, q.QueryName, q.Mute)
		if err != nil {
			return fmt.Errorf("error while querying PRs %s: %w", q.GitHubArg, err)
		}
		for _, pr := range prs {
			if queriedPrs[pr.URL] == nil {
				queriedPrs[pr.URL] = []gh.PullRequest{}
			}
			queriedPrs[pr.URL] = append(queriedPrs[pr.URL], pr)
		}
	}
	log.Printf("Got %d PRs (with duplicates)", len(queriedPrs))
	log.Printf("Use attribution order: %s", strings.Join(config.AttributionOrder, ", "))
	attributionPriority := make(map[string]int)
	for i, queryName := range config.AttributionOrder {
		attributionPriority[queryName] = i
	}
	// PRs that are not duplicated anymore, with attribution w.r.t. attribution order.
	uniquePrs := []gh.PullRequest{}
	for _, prs := range queriedPrs {
		if len(prs) == 1 {
			uniquePrs = append(uniquePrs, prs[0])
		} else {
			selected := selectPrWrtAttributionPriority(prs, attributionPriority)
			uniquePrs = append(uniquePrs, selected)
		}
	}

	if err := s.Storage.ResetPullRequests(uniquePrs); err != nil {
		return fmt.Errorf("error while storing PRs: %w", err)
	}

	log.Printf("Updated %d pull requests", len(uniquePrs))
	return nil
}

func selectPrWrtAttributionPriority(prs []gh.PullRequest, attributionPriority map[string]int) gh.PullRequest {
	selected := prs[0]
	for _, pr := range prs {
		if attributionPriority[pr.Meta.Label] < attributionPriority[selected.Meta.Label] {
			selected = pr
		}
	}
	return selected
}

func getPrs(query, metaLabel string, mute bool) ([]gh.PullRequest, error) {
	log.Printf("Get PRs for: %s", query)
	out, err := exec.Command("gh", "search", "prs", "--draft=false", "--state=open", query, "--json", jsonFields).Output()
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
		pr.Meta.DefaultMute = mute
		prs[i] = pr
	}
	return prs, nil
}
