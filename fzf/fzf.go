package fzf

import (
	"cmp"
	"ffgh/config"
	"ffgh/gh"
	"ffgh/ghutil"
	"ffgh/storage"
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
)

const nbsp = "\u00A0"

// FprintPullRequests prints the list of pull requests, one per line, in a format suitable for consumption by `fzf`.
func FprintPullRequests(out io.Writer, terminalWidth int, prs []gh.PullRequest, userState *storage.UserState, config config.Config) {
	log.Printf("Use terminal width of %d", terminalWidth)
	isMute := func(pr gh.PullRequest) bool {
		return ghutil.IsMute(userState, pr)
	}

	useMuted := func(prs []gh.PullRequest) []gh.PullRequest {
		filtered := []gh.PullRequest{}
		for _, pr := range prs {
			if isMute(pr) {
				filtered = append(filtered, pr)
			}
		}
		return filtered
	}

	useNotMuted := func(prs []gh.PullRequest) []gh.PullRequest {
		filtered := []gh.PullRequest{}
		for _, pr := range prs {
			if !isMute(pr) {
				filtered = append(filtered, pr)
			}
		}
		return filtered
	}

	displayPriority := make(map[string]int)
	for i, queryName := range config.DisplayOrder {
		displayPriority[queryName] = i
	}
	slices.SortStableFunc(prs, func(a, b gh.PullRequest) int {
		return cmp.Compare(a.Number, b.Number)
	})

	slices.SortStableFunc(prs, func(a, b gh.PullRequest) int {
		return cmp.Compare(a.Repository.Name, b.Repository.Name)
	})

	slices.SortStableFunc(prs, func(a, b gh.PullRequest) int {
		return cmp.Compare(displayPriority[a.Meta.Label], displayPriority[b.Meta.Label])
	})

	if mode := userState.Settings.ViewMode; mode == ViewModeMuteTop {
		newPrs := append([]gh.PullRequest{}, useNotMuted(prs)...)
		prs = append(newPrs, useMuted(prs)...)
	} else if mode == ViewModeHideMute {
		prs = useNotMuted(prs)
	}

	repoNameMaxLen := getMaxRepoLen(prs)
	for _, pr := range prs {
		prState := userState.PerUrl[pr.URL]
		flagString := ""
		flags := storage.GetPrStateFlags(pr, prState)
		log.Printf("Flags for %s: b%b", pr.URL, flags)
		mute := isMute(pr)
		unmutedOnly := func(c func(string, ...any) string, s string) string {
			if mute {
				return s
			} else {
				return c(s)
			}
		}
		if flags&storage.IS_NEW != 0 {
			flagString += unmutedOnly(color.GreenString, "N")
		} else {
			flagString += nbsp
		}
		if flags&storage.IS_UPDATED != 0 {
			flagString += unmutedOnly(color.HiWhiteString, "U")
		} else {
			flagString += nbsp
		}
		if flags&storage.HAS_NEW_COMMENTS != 0 {
			flagString += unmutedOnly(color.HiYellowString, "C")
		} else {
			flagString += nbsp
		}
		title := pr.Title
		note := ""
		if prState.Note != "" {
			//title = title + unmutedOnly(color.CyanString, " ["+prState.Note+"]")
			note = unmutedOnly(color.CyanString, " ["+prState.Note+"]")
		}

		shortLabel := " "
		for _, q := range config.Queries {
			if q.QueryName == pr.Meta.Label {
				shortLabel = q.ShortName
			}
		}
		leftParts := []string{
			flagString,
			toLeftS(pr.Repository.Name, repoNameMaxLen),
			shortLabel,
			fmt.Sprintf("#%-5d", pr.Number),
			title,
		}
		lineLeft := strings.Join(leftParts, " ")
		lineRight := note
		line := fmt.Sprintf("%s\t%s", pr.URL, joinStringsCapWidth(terminalWidth, lineLeft, lineRight))
		if mute {
			line = color.HiBlackString(line)
		}
		fmt.Fprint(out, line+"\n")
	}
}

func FprintShowPullRequest(out io.Writer, prUrl string, prs []gh.PullRequest, userPrState *storage.UserState) {
	var pr *gh.PullRequest
	for i := range prs {
		if prs[i].URL == prUrl {
			pr = &prs[i]
			break
		}
	}
	if pr == nil {
		// no such pr
		return
	}
	prState := userPrState.PerUrl[pr.URL]
	note := ""
	if prState.Note != "" {
		note = color.YellowString("[" + prState.Note + "]")
	}
	flags := storage.GetPrStateFlags(*pr, prState)
	flagString := ""

	if flags&storage.IS_NEW != 0 {
		flagString += color.GreenString("NEW ")
	}
	if flags&storage.IS_UPDATED != 0 {
		flagString += color.HiWhiteString("UPDATED ")
	}
	if flags&storage.HAS_NEW_COMMENTS != 0 {
		flagString += color.HiYellowString("COMMENTS")
	}

	now := time.Now()
	details := []string{
		color.HiRedString(pr.Repository.NameWithOwner),
		color.CyanString(
			fmt.Sprintf("(#%d) %s", pr.Number, pr.Title),
		),
		"",
		flagString,
		color.YellowString(fmt.Sprintf("%s (%s)", pr.Author.Login, pr.Meta.Label)),
		color.YellowString(fmt.Sprintf("Created %s, updated %s ago",
			PrettyDuration(now.Sub(pr.CreatedAt).Round(time.Minute)),
			PrettyDuration(now.Sub(pr.UpdatedAt).Round(time.Minute)),
		)),
		color.YellowString(fmt.Sprintf("%d comment(s)", pr.CommentsCount)),
		note,
		"",
		pr.Body,
	}
	fmt.Fprint(out, strings.Join(details, "\n"))
}

func getMaxRepoLen(prs []gh.PullRequest) int {
	repoNameMaxLen := 0
	for _, pr := range prs {
		if l := len(pr.Repository.Name); l > repoNameMaxLen {
			repoNameMaxLen = l
		}
	}
	return repoNameMaxLen
}

func toLeftS(s string, w int) string {
	f := fmt.Sprintf("%%-%ds", w) // like %-10s
	return fmt.Sprintf(f, s)
}

const elypsis = '…'

func joinStringsCapWidth(width int, left, right string) string {
	leftSize := utf8.RuneCountInString(left)
	rightSize := utf8.RuneCountInString(right)
	if leftSize+rightSize <= width {
		return left + right
	}
	buf := make([]rune, width)
	i := 0
	for _, r := range left {
		if i >= len(buf) {
			break
		}
		buf[i] = r
		i++
	}
	i = len(buf) - rightSize
	if j := (i - 1); j >= 0 {
		buf[j] = elypsis
	}
	for _, r := range right {
		if i >= len(buf) {
			break
		}
		buf[i] = r
		i++
	}
	return string(buf)
}
