package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/benweidig/tortuga/internal/discovery"
	"github.com/benweidig/tortuga/internal/git"
	"github.com/benweidig/tortuga/internal/model"
	"github.com/benweidig/tortuga/internal/ui"
	"github.com/benweidig/tortuga/version"
)

type Config struct {
	Dir        string
	Monochrome bool
	AutoYes    bool
	AutoNo     bool
	Jobs       int
}

func parseConfig(args []string) (Config, error) {
	// Pre-process args: replace --foo with -foo so the standard flag package
	// accepts both short (-m) and long (--monochrome) forms.
	for i, arg := range args {
		if strings.HasPrefix(arg, "--") {
			args[i] = arg[1:]
		}
	}

	fs := flag.NewFlagSet("tt", flag.ContinueOnError)

	var cfg Config
	fs.BoolVar(&cfg.Monochrome, "m", false, "")
	fs.BoolVar(&cfg.Monochrome, "monochrome", false, "disable colors")
	fs.BoolVar(&cfg.AutoYes, "y", false, "")
	fs.BoolVar(&cfg.AutoYes, "yes", false, "skip prompt and sync")
	fs.BoolVar(&cfg.AutoNo, "n", false, "")
	fs.BoolVar(&cfg.AutoNo, "no", false, "fetch only, no changes")
	fs.IntVar(&cfg.Jobs, "j", 5, "")
	fs.IntVar(&cfg.Jobs, "jobs", 5, "max concurrent git operations")

	var showVersion bool
	fs.BoolVar(&showVersion, "version", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "tt %s\n\n", version.BuildVersion())
		fmt.Fprintf(os.Stderr, "Usage: tt [flags] [directory]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  -m, --monochrome  disable colors\n")
		fmt.Fprintf(os.Stderr, "  -y, --yes         skip prompt and sync\n")
		fmt.Fprintf(os.Stderr, "  -n, --no          fetch only, no changes\n")
		fmt.Fprintf(os.Stderr, "  -j, --jobs N      max concurrent git operations (default 5)\n")
		fmt.Fprintf(os.Stderr, "  -h, --help        show this help\n")
		fmt.Fprintf(os.Stderr, "      --version     print version and exit\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cfg, err
		}

		return cfg, fmt.Errorf("flag: %w", err)
	}

	if showVersion {
		fmt.Printf("tt %s\n", version.BuildVersion())
		return cfg, flag.ErrHelp // reuse ErrHelp as "exit cleanly" signal
	}

	if cfg.AutoYes && cfg.AutoNo {
		return cfg, fmt.Errorf("-y/--yes and -n/--no are mutually exclusive")
	}

	if cfg.Jobs < 1 {
		return cfg, fmt.Errorf("--jobs must be at least 1")
	}

	if fs.NArg() > 0 {
		cfg.Dir = fs.Arg(0)
	} else {
		wd, err := os.Getwd()

		if err != nil {
			return cfg, fmt.Errorf("could not determine working directory: %w", err)
		}

		cfg.Dir = wd
	}

	return cfg, nil
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}

		fmt.Fprintf(os.Stderr, "tt: %s\n", err)
		os.Exit(2)
	}

	if err := git.IsAvailable(); err != nil {
		fmt.Fprintln(os.Stderr, "tt: git not found in PATH")
		os.Exit(2)
	}

	repos, err := discovery.FindRepos(cfg.Dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tt: %s\n", err)
		os.Exit(2)
	}

	sem := make(chan struct{}, cfg.Jobs)
	writer := ui.NewStdoutWriter()
	renderer := ui.NewRenderer(writer, cfg.Monochrome)

	setupSignalHandler()

	repos, _ = ui.RunFetchPhase(repos, sem, renderer)

	action := resolveAction(cfg, repos)

	if action == ui.ActionNone {
		os.Exit(exitCode(repos))
	}

	filtered := ui.FilterRepos(repos, action)
	if len(filtered) == 0 {
		fmt.Println("Nothing to do.")
		os.Exit(0)
	}

	renderer.StartFresh()

	fmt.Println()

	ui.RunSyncPhase(filtered, action, sem, renderer)

	if anyErrors(repos) || anyErrors(filtered) {
		os.Exit(1)
	}

	os.Exit(0)
}

// resolveAction determines what the user wants to do: honour flags first, then
// check whether stdout is a TTY (piped output implies -n), then prompt.
func resolveAction(cfg Config, repos []model.Repo) ui.Action {
	if cfg.AutoNo {
		return ui.ActionNone
	}

	if cfg.AutoYes {
		return ui.ActionSync
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return ui.ActionNone
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return ui.ActionNone
	}
	defer tty.Close()

	action, err := ui.AskAction(tty, repos)
	if err != nil {
		return ui.ActionNone
	}

	return action
}

// setupSignalHandler ensures a clean exit on SIGINT/SIGTERM. A newline is
// printed so the shell prompt appears on a fresh line even if the signal fires
// mid-render or during the raw-mode prompt.
func setupSignalHandler() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Fprintln(os.Stdout)
		os.Exit(1)
	}()
}

func anyErrors(repos []model.Repo) bool {
	for _, r := range repos {
		if r.Err != nil {
			return true
		}
	}

	return false
}

func exitCode(repos []model.Repo) int {
	if anyErrors(repos) {
		return 1
	}

	return 0
}
