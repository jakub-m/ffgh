package fzf

import (
	"fmt"
	"strings"
	"time"
)

type PrettyDuration time.Duration

func (p PrettyDuration) String() string {
	d := time.Duration(p)
	daysTime := d.Truncate(time.Hour * 24)
	days := int(daysTime / time.Hour / 24)
	cut := func(s string) string {
		b, _ := strings.CutSuffix(s, "0s")
		return b
	}
	if days == 0 {
		return cut(d.String())
	}
	return cut(fmt.Sprintf("%dd%s", days, d-daysTime))
}
