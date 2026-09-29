package term

import "github.com/gdamore/tcell/v2"

// EGA colors used by Norton Commander. tcell downgrades them to the nearest
// palette colors on terminals without true color.
var (
	Black      = tcell.NewRGBColor(0x00, 0x00, 0x00)
	Blue       = tcell.NewRGBColor(0x00, 0x00, 0xAA)
	Cyan       = tcell.NewRGBColor(0x00, 0xAA, 0xAA)
	Red        = tcell.NewRGBColor(0xAA, 0x00, 0x00)
	LightGray  = tcell.NewRGBColor(0xAA, 0xAA, 0xAA)
	BrightCyan = tcell.NewRGBColor(0x55, 0xFF, 0xFF)
	Yellow     = tcell.NewRGBColor(0xFF, 0xFF, 0x55)
	White      = tcell.NewRGBColor(0xFF, 0xFF, 0xFF)
)

func style(fg, bg tcell.Color) tcell.Style {
	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// Color roles.
var (
	PanelStyle          = style(BrightCyan, Blue)
	HeaderStyle         = style(Yellow, Blue)
	CursorStyle         = style(Black, Cyan)
	SelectedStyle       = style(Yellow, Blue)
	SelectedCursorStyle = style(Yellow, Cyan)
	ActiveTitleStyle    = style(Black, Cyan)
	CmdLineStyle        = style(LightGray, Black)
	KeyNumStyle         = style(LightGray, Black)
	KeyLabelStyle       = style(Black, Cyan)
	ErrorStyle          = style(White, Red)
	DialogStyle         = style(Black, LightGray)
	DialogFrameStyle    = style(White, LightGray)
	ButtonStyle         = style(Black, White)
	ButtonFocusStyle    = style(Black, Yellow)
	InputStyle          = style(Black, Cyan)
	MenuStyle           = style(Black, Cyan)
	MenuSelStyle        = style(White, Black)
	ShadowStyle         = style(Black, Black)
)

var styleCodes = map[[2]tcell.Color]rune{}

func init() {
	for _, sc := range []struct {
		st   tcell.Style
		code rune
	}{
		{PanelStyle, 'p'},
		{HeaderStyle, 'y'}, // also SelectedStyle
		{CursorStyle, 'c'}, // also ActiveTitleStyle, KeyLabelStyle, InputStyle, MenuStyle
		{SelectedCursorStyle, 'Y'},
		{CmdLineStyle, 'k'}, // also KeyNumStyle
		{ErrorStyle, 'r'},
		{DialogStyle, 'g'},
		{DialogFrameStyle, 'G'},
		{ButtonStyle, 'b'},
		{ButtonFocusStyle, 'B'},
		{MenuSelStyle, 'w'},
		{ShadowStyle, 'x'},
		{tcell.StyleDefault, '.'},
	} {
		fg, bg, _ := sc.st.Decompose()
		styleCodes[[2]tcell.Color{fg, bg}] = sc.code
	}
}

// StyleCode returns a one-letter code of the color role of st, used by
// golden screen dumps. Unknown combinations give '?'.
func StyleCode(st tcell.Style) rune {
	fg, bg, _ := st.Decompose()
	if c, ok := styleCodes[[2]tcell.Color{fg, bg}]; ok {
		return c
	}
	return '?'
}
