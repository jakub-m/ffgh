package xbar

import (
	"ffgh/gh"
	"ffgh/storage"
	"fmt"
	"io"
	"strings"
)

func FprintCompactSummary(out io.Writer, prs []gh.PullRequest, userPrState *storage.UserState) {
	newCount := 0
	updatedCount := 0
	commentedCount := 0
	totalCount := 0
	for _, pr := range prs {
		prState := userPrState.Get(pr.URL)
		if prState.IsMute {
			continue
		}
		totalCount++
		flags := storage.GetPrStateFlags(pr, prState)
		switch {
		case flags&storage.IS_NEW != 0:
			newCount++
		case flags&storage.IS_UPDATED != 0:
			updatedCount++
		case flags&storage.HAS_NEW_COMMENTS != 0:
			commentedCount++
		}
	}
	parts := []string{}
	parts = append(parts, fmt.Sprintf("GH%d", totalCount))
	if newCount > 0 {
		parts = append(parts, fmt.Sprintf("N%d", newCount))
	}
	if updatedCount > 0 {
		parts = append(parts, fmt.Sprintf("U%d", updatedCount))
	}
	if commentedCount > 0 {
		parts = append(parts, fmt.Sprintf("C%d", commentedCount))
	}
	fmt.Fprint(out, strings.Join(parts, ":"))
}
