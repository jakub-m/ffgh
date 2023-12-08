package main

import (
	"ffgh/config"
	conf "ffgh/config"
	"ffgh/fzf"
	"ffgh/gh"
	"ffgh/storage"
	"ffgh/sync"
	"ffgh/xbar"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	commandAddNote            = "add-note"
	commandCycleView          = "cycle-view"
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
		fmt.Printf("Available commands: %s\n", strings.Join(commands, ", "))
		fmt.Printf("\n")
		flag.PrintDefaults()
		fmt.Printf("\nThe default config file is:\n%s", config.DefaultConfigYaml)
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
		} else {
			return fmt.Errorf("unknown command: %s", command)
		}
	}(); err != nil {
		log.Fatalf("Error for command %s: %s", command, err)
	}
}

func runCommandSync(config config.Config, storage storage.Storage) error {
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

func runCommandFzf(config config.Config, storage storage.Storage) error {
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
	fzf.FprintPullRequests(out, prs, userState, config.Queries)
	return nil
}

func runCommandShowCompactSummary(storage storage.Storage) error {
	prs, userPrState, err := loadState(storage)
	if err != nil {
		return fmt.Errorf("storage failed: %w", err)
	}
	outOfSyncTime := time.Now().Add(-outOfSyncPeriod)
	if syncTime, ok := storage.GetSyncTime(); !ok || syncTime.Before(outOfSyncTime) {
		fmt.Printf("GH err!")
	} else {
		xbar.FprintCompactSummary(os.Stdout, prs, userPrState)
	}
	return nil
}

func runCommandShowPr(storage storage.Storage) error {
	if len(flag.Args()) < 2 {
		return fmt.Errorf("expected url to identify pr")
	}
	prUrl := flag.Args()[1]
	prs, userPrState, err := loadState(storage)
	if err != nil {
		return fmt.Errorf("storage failed: %w", err)
	}
	fzf.FprintShowPullRequest(os.Stdout, prUrl, prs, userPrState)
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
		return fmt.Errorf("expected url to mark and file with note")
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

func loadState(storage storage.Storage) ([]gh.PullRequest, *storage.UserState, error) {
	prs, err := storage.GetPullRequests()
	if err != nil {
		return prs, nil, err
	}
	userPrState, err := storage.GetUserState()
	if err != nil {
		return prs, nil, err
	}
	return prs, userPrState, nil
}

func getDefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("error on UserHomeDir: %s", err)
		home = ""
	}
	return path.Join(home, ".ffgh")
}
