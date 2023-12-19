package fzf

import "ffgh/util"

const (
	ViewModeRegular  = "regular"
	ViewModeMuteTop  = "mute-top"
	ViewModeHideMute = "hide-mute"
)

var viewModes = []string{
	ViewModeRegular,
	ViewModeMuteTop,
	ViewModeHideMute,
}

func CycleViewMode(m string) string {
	return util.Cycle(m, viewModes)
}
