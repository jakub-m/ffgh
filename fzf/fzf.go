package fzf

import (
	"cmp"
	"ffgh/config"
	"ffgh/gh"
	"ffgh/storage"
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
)

const nbsp = "\u00A0"

func FprintPullRequests(out io.Writer, prs []gh.PullRequest, userState *storage.UserState, config config.Config) {
	useMuted := func(prs []gh.PullRequest) []gh.PullRequest {
		filtered := []gh.PullRequest{}
		for _, pr := range prs {
			if userState.GetPR(pr.URL).IsMute {
				filtered = append(filtered, pr)
			}
		}
		return filtered
	}

	useNotMuted := func(prs []gh.PullRequest) []gh.PullRequest {
		filtered := []gh.PullRequest{}
		for _, pr := range prs {
			if !userState.GetPR(pr.URL).IsMute {
				filtered = append(filtered, pr)
			}
		}
		return filtered
	}

	displayPriority := make(map[string]int)
	for i, queryName := range config.DisplayOrder {
		displayPriority[queryName] = i
	}
	if mode := userState.Settings.ViewMode; mode == ViewModeMuteTop {
		newPrs := append([]gh.PullRequest{}, useNotMuted(prs)...)
		newPrs = append(newPrs, useMuted(prs)...)
		prs = newPrs

	} else if mode == ViewModeHideMute {
		prs = useNotMuted(prs)
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

	repoNameMaxLen := getMaxRepoLen(prs)
	for _, pr := range prs {
		prState := userState.GetPR(pr.URL)
		flagString := ""
		flags := storage.GetPrStateFlags(pr, prState)
		log.Printf("Flags for %s: b%b", pr.URL, flags)
		mute := prState.IsMute
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
		if prState.Note != "" {
			title = title + unmutedOnly(color.CyanString, " ["+prState.Note+"]")
		}

		shortLabel := " "
		for _, q := range config.Queries {
			if q.QueryName == pr.Meta.Label {
				shortLabel = q.ShortName
			}
		}
		outputParts := []string{
			flagString,
			toLeftS(pr.Repository.Name, repoNameMaxLen),
			shortLabel,
			fmt.Sprintf("#%-5d", pr.Number),
			title,
		}
		line := fmt.Sprintf("%s\t%s", pr.URL, strings.Join(outputParts, " "))
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
	prState := userPrState.GetPR(pr.URL)
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
