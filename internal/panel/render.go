package panel

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/navoznov/terminal-commander/internal/fs"
	"github.com/navoznov/terminal-commander/internal/term"
)

const (
	sizeW = 9
	dateW = 8
	timeW = 6
	// Columns taken by "│size│date│time" in full mode.
	fullFixed = 1 + sizeW + 1 + dateW + 1 + timeW
)

type column struct {
	off, width int
	header     string
}

func (p *Panel) columns(inner int) []column {
	if p.Mode == Full {
		nameW := inner - fullFixed
		return []column{
			{0, nameW, "Name"},
			{nameW + 1, sizeW, "Size"},
			{nameW + 2 + sizeW, dateW, "Date"},
			{nameW + 3 + sizeW + dateW, timeW, "Time"},
		}
	}
	cw := (inner - 2) / briefCols
	return []column{
		{0, cw, "Name"},
		{cw + 1, cw, "Name"},
		{2*cw + 2, inner - 2*cw - 2, "Name"},
	}
}

// Draw renders the panel into the rectangle (x, y, w, h). The caller sets
// the list height with SetRows(h - 5) beforehand.
func (p *Panel) Draw(c term.Canvas, x, y, w, h int, active bool, home string) {
	p.x, p.y, p.w = x, y, w
	st := term.PanelStyle
	inner := w - 2
	sepY := y + h - 3
	c.Fill(x, y, w, h, ' ', st)
	c.Box(x, y, w, h, st)
	c.Put(x, sepY, '╟', st)
	c.HLine(x+1, sepY, inner, '─', st)
	c.Put(x+w-1, sepY, '╢', st)

	cols := p.columns(inner)
	for i, col := range cols {
		cx := x + 1 + col.off
		if i > 0 {
			c.Put(cx-1, y, '╤', st)
			c.VLine(cx-1, y+1, h-4, '│', st)
			c.Put(cx-1, sepY, '┴', st)
		}
		hw := term.Width(col.header)
		c.Text(cx+max(0, (col.width-hw)/2), y+1, col.header, col.width, term.HeaderStyle)
	}

	p.drawEntries(c, x+1, y+2, cols, active)
	p.drawStatus(c, x+1, y+h-2, inner)
	p.drawTitle(c, x, y, w, active, home)
}

// Hit returns the index of the entry drawn at screen cell (x, y).
func (p *Panel) Hit(x, y int) (int, bool) {
	r := y - p.y - 2
	if r < 0 || r >= p.rows || x <= p.x || x >= p.x+p.w-1 {
		return 0, false
	}
	i := p.Top + r
	if p.Mode == Brief {
		k := -1
		for j, col := range p.columns(p.w - 2) {
			if cx := p.x + 1 + col.off; x >= cx && x < cx+col.width {
				k = j
			}
		}
		if k < 0 {
			return 0, false // a column line
		}
		i += k * p.rows
	}
	if i >= len(p.Entries) {
		return 0, false
	}
	return i, true
}

func (p *Panel) drawEntries(c term.Canvas, x0, y0 int, cols []column, active bool) {
	if p.Mode == Full {
		for r := 0; r < p.rows; r++ {
			i := p.Top + r
			if i >= len(p.Entries) {
				return
			}
			e := p.Entries[i]
			st := p.entryStyle(i, active)
			fields := []string{
				fitName(e, cols[0].width),
				fitRight(p.sizeText(e), sizeW),
				fitRight(fs.FormatDate(e.ModTime), dateW),
				fitRight(fs.FormatTime(e.ModTime), timeW),
			}
			for j, col := range cols {
				c.Text(x0+col.off, y0+r, fields[j], col.width, st)
				if j > 0 && active && i == p.Cursor {
					c.Put(x0+col.off-1, y0+r, '│', st)
				}
			}
		}
		return
	}
	for k := 0; k < p.rows*len(cols); k++ {
		i := p.Top + k
		if i >= len(p.Entries) {
			return
		}
		col := cols[k/p.rows]
		c.Text(x0+col.off, y0+k%p.rows, fitName(p.Entries[i], col.width), col.width, p.entryStyle(i, active))
	}
}

func (p *Panel) entryStyle(i int, active bool) tcell.Style {
	sel := p.Selected[p.Entries[i].Name]
	cur := active && i == p.Cursor
	switch {
	case cur && sel:
		return term.SelectedCursorStyle
	case cur:
		return term.CursorStyle
	case sel:
		return term.SelectedStyle
	}
	return term.PanelStyle
}

func (p *Panel) drawStatus(c term.Canvas, x, y, w int) {
	if n, bytes := p.SelectionStats(); n > 0 {
		size := fs.FormatThousands(bytes) + " bytes"
		if p.HumanSizes && bytes >= 1024 {
			size = fs.FormatUnits(bytes)
		}
		s := fmt.Sprintf("%s in %d selected files", size, n)
		c.Text(x+max(0, (w-term.Width(s))/2), y, s, w, term.SelectedStyle)
		return
	}
	e := p.Current()
	if e == nil {
		return
	}
	right := fitRight(p.sizeText(*e), sizeW) + " " +
		fitRight(fs.FormatDate(e.ModTime), dateW) + " " +
		fitRight(fs.FormatTime(e.ModTime), timeW)
	nameW := w - term.Width(right) - 1
	c.Text(x, y, term.Fit(e.Name, nameW)+" "+right, w, term.PanelStyle)
}

func (p *Panel) drawTitle(c term.Canvas, x, y, w int, active bool, home string) {
	t := fs.DisplayPath(p.Path, home)
	if limit := w - 4; term.Width(t) > limit {
		t = "…" + term.Tail(t, limit-1)
	}
	t = " " + t + " "
	st := term.PanelStyle
	if active {
		st = term.ActiveTitleStyle
	}
	tw := term.Width(t)
	c.Text(x+(w-tw)/2, y, t, tw, st)
}

// fitName formats a name for a column of width w. Files with a 1–3 column
// extension get the DOS layout: name, then the extension in the last three
// columns ("autoexec bat"). Everything else is shown whole, cut with '}'.
func fitName(e fs.Entry, w int) string {
	if !e.IsDir && w >= 5 {
		if base, ext, ok := splitExt(e.Name); ok {
			return term.Fit(base, w-4) + " " + term.Fit(ext, 3)
		}
	}
	return term.Fit(e.Name, w)
}

func splitExt(name string) (base, ext string, ok bool) {
	i := strings.LastIndexByte(name, '.')
	if i <= 0 || i == len(name)-1 {
		return "", "", false
	}
	if term.Width(name[i+1:]) > 3 {
		return "", "", false
	}
	return name[:i], name[i+1:], true
}

func (p *Panel) sizeText(e fs.Entry) string {
	switch {
	case e.IsUp:
		return "►UP--DIR◄"
	case e.IsDir:
		return "►SUB-DIR◄"
	}
	if p.HumanSizes {
		return fs.FormatUnits(e.Size)
	}
	return fs.FormatSize(e.Size, sizeW)
}

func fitRight(s string, w int) string {
	if sw := term.Width(s); sw < w {
		return strings.Repeat(" ", w-sw) + s
	}
	return term.Fit(s, w)
}
