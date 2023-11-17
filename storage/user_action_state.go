package storage

import (
	"encoding/json"
	"ffgh/gh"
	"time"
)

type UserPrState struct {
	perUrl map[string]PrState
}

type userPrStateJson struct {
	PerUrl map[string]PrState
}

func (s *UserPrState) MarshalJSON() ([]byte, error) {
	q := userPrStateJson{
		PerUrl: s.perUrl,
	}
	return json.Marshal(q)
}

func (s *UserPrState) UnmarshalJSON(b []byte) error {
	var q userPrStateJson
	if err := json.Unmarshal(b, &q); err != nil {
		return err
	}
	s.perUrl = q.PerUrl
	return nil
}

var _ json.Marshaler = (*UserPrState)(nil)
var _ json.Unmarshaler = (*UserPrState)(nil)

func (s *UserPrState) Get(url string) PrState {
	if state, ok := s.perUrl[url]; ok {
		return state
	} else {
		state := PrState{}
		s.perUrl[url] = state
		return state
	}
}

func (s *UserPrState) Set(url string, p PrState) {
	s.perUrl[url] = p
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
