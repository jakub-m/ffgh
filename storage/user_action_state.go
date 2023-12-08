package storage

import (
	"ffgh/gh"
	"time"
)

type UserState struct {
	PerUrl map[string]PrState
}

func (s *UserState) Get(url string) PrState {
	if state, ok := s.PerUrl[url]; ok {
		return state
	} else {
		state := PrState{}
		s.PerUrl[url] = state
		return state
	}
}

func (s *UserState) Set(url string, p PrState) {
	s.PerUrl[url] = p
}

type PrState struct {
	OpenedAt         *time.Time
	LastCommentCount int
	Note             string
	IsMute           bool
}

const (
	HAS_NEW_COMMENTS = 1 << iota
	IS_UPDATED
	IS_NEW
)

func GetPrStateFlags(pr gh.PullRequest, prState PrState) int {
	out := 0
	if pr.CommentsCount > prState.LastCommentCount {
		out |= HAS_NEW_COMMENTS
	}
	if prState.OpenedAt == nil {
		out |= IS_NEW
	} else if pr.UpdatedAt.After(*prState.OpenedAt) {
		out |= IS_UPDATED
	}
	return out
}
