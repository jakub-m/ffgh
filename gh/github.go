package gh

import "time"

type Author struct {
	ID    string `json:"id"`
	IsBot bool   `json:"is_bot"`
	Login string `json:"login"`
	Type  string `json:"type"`
	URL   string `json:"url"`
}

type Repository struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

type PullRequest struct {
	Author        Author     `json:"author"`
	Body          string     `json:"body"`
	CommentsCount int        `json:"commentsCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	ID            string     `json:"id"`
	Number        int        `json:"number"`
	Repository    Repository `json:"repository"`
	Title         string     `json:"title"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	URL           string     `json:"url"`
	State         string     `json:"state"`
	Meta          Meta       `json:"_meta"`
}

// Meta is metadata attached to PR that is not a part of the GitHub payload.
type Meta struct {
	Label string
}
