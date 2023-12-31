package storage

import (
	"encoding/json"
	"ffgh/gh"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	defaultGitHubState = "gh_daemon_state.json"
	defaultUserState   = "gh_user_state.json"
)

func NewFileStorage() *FileStorage {
	return &FileStorage{
		PrsStatePath:  defaultGitHubState,
		UserStatePath: defaultUserState,
	}
}

type FileStorage struct {
	PrsStatePath  string
	UserStatePath string
}

var _ Storage = (*FileStorage)(nil)

func (s *FileStorage) ResetPullRequests(prs []gh.PullRequest) error {
	marshalled, err := json.MarshalIndent(prs, "", " ")
	if err != nil {
		return fmt.Errorf("error while marshalling PRs: %w", err)
	}
	return writeAtOnce(s.PrsStatePath, marshalled)
}

func (s *FileStorage) MarkUrlAsOpened(url string) (bool, error) {
	log.Printf("Mark opened %s", url)
	pr, err := s.getPrForUrl(url)
	if err != nil {
		return false, fmt.Errorf("error when marking open: %w", err)
	}
	userPrState, err := s.readUserState()
	if err != nil {
		return false, fmt.Errorf("error while reading user state: %w", err)
	}
	prState := userPrState.GetPR(url)
	if prState.OpenedAt != nil && *prState.OpenedAt == pr.UpdatedAt && prState.LastCommentCount == pr.CommentsCount {
		log.Printf("PR state up to date, not marking it as opened")
		return false, nil
	} else {
		log.Printf("PR state changed so it's marked as opened")
		prState.OpenedAt = &pr.UpdatedAt
		prState.LastCommentCount = pr.CommentsCount
		userPrState.Set(url, prState)
		return true, s.writeUserState(userPrState)
	}
}

func (s *FileStorage) MarkUrlAsMuted(url string) error {
	log.Printf("Mark muted %s", url)
	// mark as read
	userPrState, err := s.readUserState()
	if err != nil {
		return fmt.Errorf("error while reading user state: %w", err)
	}
	prState := userPrState.GetPR(url)
	// If already read, then change mute state
	prState.IsMute = !prState.IsMute
	log.Printf("Change mute state to %t %s", prState.IsMute, url)
	userPrState.Set(url, prState)
	return s.writeUserState(userPrState)
}

func (s *FileStorage) getPrForUrl(url string) (gh.PullRequest, error) {
	prs, err := s.GetPullRequests()
	if err != nil {
		return gh.PullRequest{}, fmt.Errorf("error when marking url %s: %w", url, err)
	}

	for _, pr := range prs {
		if pr.URL == url {
			return pr, nil
		}
	}
	return gh.PullRequest{}, fmt.Errorf("no such pr with url: %s", url)
}

func (s *FileStorage) AddNote(url, note string) error {
	log.Printf("Add note to URL %s: %s", url, note)
	userPrState, err := s.readUserState()
	if err != nil {
		return fmt.Errorf("error when adding note: %w", err)
	}
	prState := userPrState.GetPR(url)
	prState.Note = note
	userPrState.Set(url, prState)
	return s.writeUserState(userPrState)
}

func (s *FileStorage) GetPullRequests() ([]gh.PullRequest, error) {
	log.Printf("Read %s", s.PrsStatePath)
	b, err := os.ReadFile(s.PrsStatePath)
	if err != nil {
		return nil, fmt.Errorf("error while reading %s: %w", s.PrsStatePath, err)
	}
	var prs []gh.PullRequest
	if err := json.Unmarshal(b, &prs); err != nil {
		return nil, fmt.Errorf("erorr while unmarshalling file %s: %w", s.PrsStatePath, err)
	}
	return prs, nil
}

func (s *FileStorage) GetSyncTime() (time.Time, bool) {
	info, err := os.Stat(s.PrsStatePath)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

func (s *FileStorage) GetUserState() (*UserState, error) {
	return s.readUserState()
}

func (s *FileStorage) readUserState() (*UserState, error) {
	state := UserState{PerUrl: make(map[string]PrState)}
	log.Printf("Reading %s", s.UserStatePath)
	file, err := os.Open(s.UserStatePath)
	if err != nil {
		log.Printf("Error while reading %s: %s", s.UserStatePath, err)
		return &state, nil
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("error while unmarshalling file %s: %w", s.UserStatePath, err)
	}
	return &state, nil
}

func (s *FileStorage) WriteUserState(state *UserState) error {
	return s.writeUserState(state)
}

func (s *FileStorage) writeUserState(state *UserState) error {
	// new state if missing
	marshalled, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return fmt.Errorf("error while marshalling state: %w", err)
	}
	return writeAtOnce(s.UserStatePath, marshalled)
}

// writeAtOnce tries to write as atomically as it can.
func writeAtOnce(target string, b []byte) error {
	log.Printf("Write to %s", target)
	temp := target + ".temp"
	os.Remove(temp)
	err := os.WriteFile(temp, b, 0644)
	if err != nil {
		return fmt.Errorf("error while writing to file %s: %w", target, err)
	}
	os.Remove(target)
	if err := os.Rename(temp, target); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", temp, target, err)
	}
	log.Printf("Wrote to %s", target)
	return nil
}
