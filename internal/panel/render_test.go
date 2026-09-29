package panel

import (
	"strings"
	"testing"
	"time"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
	"github.com/navoznov/terminal-commander/internal/term/termtest"
)

var day = time.Date(1994, 5, 31, 6, 22, 0, 0, time.Local)

func demo() *Panel {
	p := New()
	p.Path = "/Users/nc/GAMES"
	for _, e := range []fs.Entry{
		{Name: "..", IsDir: true, IsUp: true},
		{Name: "CIV", IsDir: true},
		{Name: "LINES", IsDir: true},
		{Name: "autoexec.bat", Size: 13},
		{Name: "command.com", Size: 54645},
		{Name: "config.sys", Size: 13},
		{Name: "read.me", Size: 577},
		{Name: "a-very-long-file-name.txt", Size: 1234567},
		{Name: "Makefile", Size: 99},
	} {
		e.ModTime = day
		p.Entries = append(p.Entries, e)
	}
	p.Selected["config.sys"] = true
	return p
}

func draw(t *testing.T, p *Panel, active bool) string {
	s := termtest.NewScreen(t, 40, 12)
	p.SetRows(12 - 5)
	p.Draw(term.Canvas{Screen: s}, 0, 0, 40, 12, active, "/Users/nc")
	return termtest.Dump(s)
}

func TestDrawBrief(t *testing.T) {
	p := demo()
	p.Focus("command.com")
	termtest.Golden(t, "brief", draw(t, p, true))
}

func TestDrawFull(t *testing.T) {
	p := demo()
	p.SetMode(Full)
	p.Focus("CIV")
	termtest.Golden(t, "full", draw(t, p, true))
}

func TestDrawInactiveHasNoCursor(t *testing.T) {
	out := draw(t, demo(), false)
	styles := strings.SplitN(out, "\n\n", 2)[1]
	if strings.ContainsAny(styles, "cY") {
		t.Fatalf("inactive panel shows cursor or active title:\n%s", out)
	}
}

func TestDrawSelectionStatus(t *testing.T) {
	out := draw(t, demo(), true)
	if !strings.Contains(out, "13 bytes in 1 selected files") {
		t.Fatalf("no selection status:\n%s", out)
	}
}

func TestDrawEmptyPanel(t *testing.T) {
	p := New()
	p.Path = "/empty"
	draw(t, p, true) // must not panic
}

func TestBriefNameLayout(t *testing.T) {
	cases := []struct {
		e    fs.Entry
		want string
	}{
		{fs.Entry{Name: "autoexec.bat"}, "autoexec bat"},
		{fs.Entry{Name: "read.me"}, "read     me "},
		{fs.Entry{Name: "Makefile"}, "Makefile    "},
		{fs.Entry{Name: ".zshrc"}, ".zshrc      "},
		{fs.Entry{Name: "archive.tar.gz"}, "archive} gz "},
		{fs.Entry{Name: "index.html"}, "index.html  "},
		{fs.Entry{Name: "my.dir", IsDir: true}, "my.dir      "},
		{fs.Entry{Name: "a-very-long-file-name.txt"}, "a-very-} txt"},
	}
	for _, c := range cases {
		if got := fitName(c.e, 12); got != c.want {
			t.Errorf("%s: got %q want %q", c.e.Name, got, c.want)
		}
	}
}

func TestBriefNFDName(t *testing.T) {
	e := fs.Entry{Name: "йод.txt"} // «йод.txt» в NFD
	if got := fitName(e, 12); got != "йод      txt" {
		t.Fatalf("got %q", got)
	}
}

func TestHit(t *testing.T) {
	p := demo()
	draw(t, p, true) // 40×12 at 0,0: 7 rows, brief columns at 1, 14, 27
	for _, tt := range []struct {
		x, y int
		i    int
		ok   bool
	}{
		{1, 2, 0, true}, {12, 8, 6, true}, {14, 2, 7, true}, {14, 3, 8, true},
		{14, 4, 0, false}, // past the last entry
		{13, 2, 0, false}, // column line
		{0, 2, 0, false},  // frame
		{1, 1, 0, false},  // header
		{1, 9, 0, false},  // status line separator
	} {
		if i, ok := p.Hit(tt.x, tt.y); i != tt.i || ok != tt.ok {
			t.Errorf("Hit(%d, %d) = %d, %v; want %d, %v", tt.x, tt.y, i, ok, tt.i, tt.ok)
		}
	}
	p.SetMode(Full)
	draw(t, p, true)
	if i, ok := p.Hit(35, 3); i != 1 || !ok {
		t.Errorf("full: Hit = %d, %v", i, ok)
	}
}
