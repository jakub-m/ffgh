package main

import (
	conf "ffgh/config"
	"ffgh/fzf"
	"ffgh/gh"
	"ffgh/storage"
	"ffgh/sync"
	"ffgh/util"
	"ffgh/xbar"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	commandAddNote            = "add-note"
	commandCycleNote          = "cycle-note"
	commandCycleView          = "cycle-view-mode"
	commandFzf                = "fzf"
	commandShowCompactSummary = "show-compact-summary"
	commandMarkOpen           = "mark-open"
	commandMarkMute           = "mark-mute"
	commandShowPr             = "show-pr"
	commandSync               = "sync"
)
const (
	// outOfSyncPeriod says how long do we wait for sync before considering the state out of sync.
	outOfSyncPeriod = 5 * time.Minute
)

func main() {
	commands := []string{
		commandAddNote,
		commandCycleNote,
		commandCycleView,
		commandFzf,
		commandMarkMute,
		commandMarkOpen,
		commandShowCompactSummary,
		commandShowPr,
		commandSync,
	}
	flag.Usage = func() {
		fmt.Printf("Utility to synchronize and display state of GitHub PRs.\n")
		fmt.Printf("Available commands: \n  - %s\n", strings.Join(commands, "\n  - "))
		fmt.Printf("\n")
		fmt.Printf("Other options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nThe default config file is:\n%s", conf.DefaultConfigYaml)
	}
	if len(os.Args) < 2 {
		flag.Usage()
		log.Fatalf("Commend missing")
	}
	options := struct {
		verbose    bool
		statePath  string
		configPath string
	}{}
	flag.BoolVar(&options.verbose, "v", false, "verbose")
	defaultStateDir := getDefaultStateDir()
	flag.StringVar(&options.statePath, "d", defaultStateDir, "directory where to store the state of the application")
	defaultConfigPath := path.Join(defaultStateDir, "config.yaml")
	flag.StringVar(&options.configPath, "c", defaultConfigPath, "config file")
	flag.Parse()
	if !options.verbose {
		log.SetOutput(io.Discard)
	}
	if options.statePath == defaultStateDir {
		log.Println("Make directory", options.statePath)
		if err := os.MkdirAll(options.statePath, 0755); err != nil {
			log.Printf("Error while making directory: %s", err)
		}
	}
	command := flag.Args()[0]
	color.NoColor = false
	config := conf.GetDefaultConfig()
	if path := options.configPath; path != "" {
		log.Printf("Read config from %s", options.configPath)
		if c, err := conf.GetConfigFromFile(path); err == nil {
			config = c
			log.Printf("Read config: %v", c)
		} else {
			log.Printf("Reading config failed, using default: %s", err)
		}
	}
	log.Printf("Run command: %s", command)
	storage := storage.NewFileStorage()
	storage.PrsStatePath = path.Join(options.statePath, storage.PrsStatePath)
	storage.UserStatePath = path.Join(options.statePath, storage.UserStatePath)
	if err := func() error {
		if command == commandSync {
			return runCommandSync(config, storage)
		} else if command == commandFzf {
			return runCommandFzf(config, storage)
		} else if command == commandShowCompactSummary {
			return runCommandShowCompactSummary(storage)
		} else if command == commandShowPr {
			return runCommandShowPr(storage)
		} else if command == commandMarkOpen {
			return runCommandMarkOpen(storage)
		} else if command == commandMarkMute {
			return runCommandMarkMute(storage)
		} else if command == commandAddNote {
			return runCommandAddNote(storage)
		} else if command == commandCycleView {
			return runCommandCycleView(storage)
		} else if command == commandCycleNote {
			return runCommandCycleNote(config, storage)
		} else {
			return fmt.Errorf("unknown command: %s", command)
		}
	}(); err != nil {
		log.Fatalf("Error for command %s: %s", command, err)
	}
}

func runCommandSync(config conf.Config, storage storage.Storage) error {
	once := false
	fs := flag.NewFlagSet(commandSync, flag.ContinueOnError)
	fs.BoolVar(&once, "once", once, "run once")
	if err := fs.Parse(flag.Args()[1:]); err != nil {
		return err
	}
	log.Printf("Run once: %t", once)
	synchronizer := sync.New()
	synchronizer.Storage = storage
	run := synchronizer.RunBlocking
	if once {
		run = synchronizer.RunOnce
	}
	return run(config)
}

func runCommandFzf(config conf.Config, storage storage.Storage) error {
	vname := "TERMINAL_WIDTH"
	terminalWidthEnv := os.Getenv(vname)
	terminalWidth, err := strconv.ParseInt(terminalWidthEnv, 10, 64)
	if err != nil {
		log.Printf("Could not figure terminal width from env variable %s: %s", vname, err)
		terminalWidth = 120
	}
	out := os.Stdout
	prs, userState, err := loadState(storage)
	if err != nil {
		return fmt.Errorf("storage failed: %w", err)
	}
	// The X is printed because all lines are displayed 2nd element.
	syncStr := ""
	if t, ok := storage.GetSyncTime(); ok {
		syncStr = fmt.Sprintf("X synced %s ago", time.Since(t).Round(time.Second))
	} else {
		syncStr = "X not synced"
	}
	fmt.Fprintf(out, "%s | %s\n", syncStr, userState.Settings.ViewMode)
	fzf.FprintPullRequests(out, int(terminalWidth), prs, userState, config)
	return nil
}

func runCommandShowCompactSummary(storage storage.Storage) error {
	prs, userState, err := loadState(storage)
	if err != nil {
		return fmt.Errorf("storage failed: %w", err)
	}
	outOfSyncTime := time.Now().Add(-outOfSyncPeriod)
	if syncTime, ok := storage.GetSyncTime(); !ok || syncTime.Before(outOfSyncTime) {
		fmt.Printf("GH err!")
	} else {
		xbar.FprintCompactSummary(os.Stdout, prs, userState)
	}
	return nil
}

func runCommandShowPr(storage storage.Storage) error {
	if len(flag.Args()) < 2 {
		return fmt.Errorf("expected url to identify pr")
	}
	prUrl := flag.Args()[1]
	prs, userState, err := loadState(storage)
	if err != nil {
		return fmt.Errorf("storage failed: %w", err)
	}
	fzf.FprintShowPullRequest(os.Stdout, prUrl, prs, userState)
	return nil
}

func runCommandMarkOpen(storage storage.Storage) error {
	fs := flag.NewFlagSet(commandMarkOpen, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("" +
			"Mark URL as opened (visited). The command returns exit code 1 if the file was visited already and the" +
			" command was no-op. This is to facilitate chaining commands with fzf bindings.")
		fs.PrintDefaults()
	}
	exitErrorIfMarked := fs.Bool("e", false, fmt.Sprintf("Exit with error code 1 if the file is already marked as opened. This is useful to conditionally chain %s and other commands.", commandMarkOpen))
	fs.Parse(flag.Args()[1:])
	log.Printf("Flag -e=%t", *exitErrorIfMarked)
	if len(fs.Args()) < 1 {
		return fmt.Errorf("expected url to mark")
	}
	url := fs.Args()[0]
	marked, err := storage.MarkUrlAsOpened(url)
	if err != nil {
		return err
	}
	if !marked && *exitErrorIfMarked {
		return fmt.Errorf("URL already marked as opened, doing nothing: %s", url)
	}
	return nil
}

func runCommandMarkMute(storage storage.Storage) error {
	if len(flag.Args()) < 2 {
		return fmt.Errorf("expected url to mark")
	}
	url := flag.Args()[1]
	return storage.MarkUrlAsMuted(url)
}

func runCommandAddNote(storage storage.Storage) error {
	if len(flag.Args()) < 3 {
		return fmt.Errorf("expected URL to mark and file with note")
	}
	url := flag.Args()[1]
	fileWithNote := flag.Args()[2]
	b, err := os.ReadFile(fileWithNote)
	if err != nil {
		return fmt.Errorf("failed to read note: %w", err)
	}
	note := strings.TrimSpace(string(b))
	return storage.AddNote(url, note)
}

func runCommandCycleView(storage storage.Storage) error {
	s, err := storage.GetUserState()
	if err != nil {
		return fmt.Errorf("error when running cycle view: %w", err)
	}
	viewMode := s.Settings.ViewMode
	s.Settings.ViewMode = fzf.CycleViewMode(viewMode)
	log.Printf("Turn view mode %s to %s", viewMode, s.Settings.ViewMode)
	if err = storage.WriteUserState(s); err != nil {
		return fmt.Errorf("error when running cycle view: %w", err)
	}
	return nil
}

func runCommandCycleNote(config conf.Config, storage storage.Storage) error {
	if len(flag.Args()) < 2 {
		return fmt.Errorf("expected URL to change note for")
	}
	if len(config.Annotations) == 0 {
		return fmt.Errorf("no annotations set in config")
	}
	url := flag.Args()[1]
	log.Printf("Cycle note for url %s", url)
	s, err := storage.GetUserState()
	if err != nil {
		return fmt.Errorf("error when running cycle view: %w", err)
	}

	currNote := ""
	if prState, ok := s.PerUrl[url]; ok {
		currNote = prState.Note
	}
	annotations := append([]string{}, config.Annotations...)
	annotations = append(annotations, "") // Add empty note at the end of the cycle
	newNote := util.Cycle(currNote, annotations)
	log.Printf("Old note '%s', new note '%s'", currNote, newNote)
	return storage.AddNote(url, newNote)
}

func loadState(storage storage.Storage) ([]gh.PullRequest, *storage.UserState, error) {
	prs, err := storage.GetPullRequests()
	if err != nil {
		return prs, nil, err
	}
	userState, err := storage.GetUserState()
	if err != nil {
		return prs, nil, err
	}
	return prs, userState, nil
}

func getDefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("error on UserHomeDir: %s", err)
		home = ""
	}
	return path.Join(home, ".ffgh")
}
