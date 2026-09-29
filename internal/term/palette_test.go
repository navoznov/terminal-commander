package term

import "testing"

func TestDialogStyleCodes(t *testing.T) {
	cases := []struct {
		name string
		code rune
		want rune
	}{
		{"dialog", StyleCode(DialogStyle), 'g'},
		{"frame", StyleCode(DialogFrameStyle), 'G'},
		{"button", StyleCode(ButtonStyle), 'b'},
		{"focus", StyleCode(ButtonFocusStyle), 'B'},
		{"input", StyleCode(InputStyle), 'c'},
		{"menu", StyleCode(MenuStyle), 'c'},
		{"menusel", StyleCode(MenuSelStyle), 'w'},
		{"shadow", StyleCode(ShadowStyle), 'x'},
	}
	for _, c := range cases {
		if c.code != c.want {
			t.Errorf("%s: got %c want %c", c.name, c.code, c.want)
		}
	}
}
