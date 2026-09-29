// Command tc is Terminal Commander, a Norton Commander look-alike for macOS.
package main

import (
	"cmp"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/app"
	"github.com/navoznov/terminal-commander/internal/config"
	"github.com/navoznov/terminal-commander/internal/keytest"
)

func main() {
	keyTest := flag.Bool("keytest", false, "show how key presses are recognised")
	flag.Parse()

	s, err := tcell.NewScreen()
	if err == nil {
		err = s.Init()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tc:", err)
		os.Exit(1)
	}
	defer func() {
		if r := recover(); r != nil {
			s.Fini()
			path := writeCrashLog(r)
			fmt.Fprintf(os.Stderr, "tc crashed: %v\ndetails: %s\n", r, path)
			os.Exit(2)
		}
	}()

	if *keyTest {
		keytest.Run(s)
		s.Fini()
		return
	}
	cwd, _ := os.Getwd()
	cfgPath := config.Path()
	cfg := config.Load(cfgPath)
	cfg.Left.Path, cfg.Right.Path = startDirs(cfg, flag.Args(), cwd)
	app.New(s, cfg, cfgPath).Run()
	s.Fini()
}

// startDirs picks the panels' directories: the saved ones, the current
// directory when none is saved, and "tc <path>" puts the left panel there.
func startDirs(cfg config.Config, args []string, cwd string) (left, right string) {
	left, right = cmp.Or(cfg.Left.Path, cwd), cmp.Or(cfg.Right.Path, cwd)
	if len(args) > 0 {
		if abs, err := filepath.Abs(args[0]); err == nil {
			left = abs
		}
	}
	return left, right
}

// writeCrashLog saves the panic and stack to
// ~/.config/terminal-commander/crash.log and returns the file path.
func writeCrashLog(r any) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "(no home directory)"
	}
	dir := filepath.Join(home, ".config", "terminal-commander")
	path := filepath.Join(dir, "crash.log")
	if os.MkdirAll(dir, 0o755) != nil {
		return "(cannot create " + dir + ")"
	}
	msg := fmt.Sprintf("panic: %v\n\n%s", r, debug.Stack())
	if os.WriteFile(path, []byte(msg), 0o644) != nil {
		return "(cannot write " + path + ")"
	}
	return path
}
