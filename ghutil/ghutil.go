package ghutil

import (
	"ffgh/gh"
	"ffgh/storage"
)

func IsMute(userState *storage.UserState, pr gh.PullRequest) bool {
	state, ok := userState.PerUrl[pr.URL]
	isMute := (ok && state.IsMute) || (!ok && pr.Meta.DefaultMute)
	return isMute
}
