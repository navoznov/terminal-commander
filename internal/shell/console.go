package shell

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// Console is the terminal while the panels are hidden. In is nil when
// there is no terminal to read from: commands get no input and nothing
// waits for a key.
type Console struct {
	In  *os.File
	Out io.Writer
}

// Run runs line with $SHELL -c (/bin/zsh if unset) in dir. Control-C stops
// the command but not tc: tc catches SIGINT and SIGQUIT while it runs.
// A caught signal, unlike an ignored one, is reset for the command.
func (c Console) Run(dir, line string) {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/zsh"
	}
	cmd := exec.Command(sh, "-c", line)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = c.Out, c.Out
	if c.In != nil {
		cmd.Stdin = c.In
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGQUIT)
	defer signal.Stop(sig)
	var exit *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &exit) {
		fmt.Fprintln(c.Out, "tc:", err)
	}
}

// Pause asks for a key and waits for it.
func (c Console) Pause() {
	fmt.Fprint(c.Out, "Press any key to continue...")
	c.WaitKey()
	fmt.Fprintln(c.Out)
}

// WaitKey waits for a key in raw mode. It reads up to 64 bytes at once so
// that the rest of an escape sequence (an arrow key) is not left for tc.
func (c Console) WaitKey() {
	if c.In == nil {
		return
	}
	fd := int(c.In.Fd())
	if !term.IsTerminal(fd) {
		return
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return
	}
	defer term.Restore(fd, old)
	var buf [64]byte
	c.In.Read(buf[:])
}
