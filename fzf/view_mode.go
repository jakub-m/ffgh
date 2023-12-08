package fzf

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
	for i, v := range viewModes {
		if v == m {
			k := (i + 1) % len(viewModes)
			return viewModes[k]
		}
	}
	return m
}
