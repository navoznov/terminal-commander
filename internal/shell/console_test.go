package shell

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInDirWithShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	dir := t.TempDir()
	var out bytes.Buffer
	if err := (Console{Out: &out}).Run(dir, "echo hi; pwd -P; echo oops >&2; exit 3"); err == nil {
		t.Fatal("exit 3 gave no error")
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := "hi\n" + real + "\noops\n"; out.String() != want {
		t.Fatalf("output %q, want %q", out.String(), want)
	}
}

func TestRunDefaultsToZsh(t *testing.T) {
	t.Setenv("SHELL", "")
	var out bytes.Buffer
	Console{Out: &out}.Run(t.TempDir(), "echo z$ZSH_VERSION")
	if strings.TrimSpace(out.String()) == "z" {
		t.Fatalf("not run by zsh: %q", out.String())
	}
}

func TestRunReportsStartError(t *testing.T) {
	t.Setenv("SHELL", "/nonexistent/sh")
	var out bytes.Buffer
	Console{Out: &out}.Run(t.TempDir(), "true")
	if !strings.HasPrefix(out.String(), "tc: ") {
		t.Fatalf("output %q", out.String())
	}
}

// If tc did not catch SIGINT while a command runs, Control-C in the
// terminal would kill it together with the command — here, the test binary.
func TestRunSurvivesInterrupt(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	var out bytes.Buffer
	Console{Out: &out}.Run(t.TempDir(), "kill -INT $PPID; sleep 0.2; echo alive")
	if out.String() != "alive\n" {
		t.Fatalf("output %q", out.String())
	}
}

func TestPauseWithoutTerminal(t *testing.T) {
	var out bytes.Buffer
	Console{Out: &out}.Pause() // must not wait
	if out.String() != "Press any key to continue...\n" {
		t.Fatalf("output %q", out.String())
	}
}
