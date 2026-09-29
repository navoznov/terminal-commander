package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCd(t *testing.T) {
	tests := []struct {
		line, dir string
		ok        bool
	}{
		{"cd", "~", true},
		{"  cd  ", "~", true},
		{"cd ..", "..", true},
		{"cd -", "-", true},
		{"cd ~/Projects", "~/Projects", true},
		{"cd   /tmp ", "/tmp", true},
		{`cd "My Dir"`, "My Dir", true},
		{`cd 'it''s'`, "its", true},
		{`cd My\ Dir`, "My Dir", true},
		{`cd "a\"b"`, `a"b`, true},
		{"cd Папка", "Папка", true},
		{"cd a b", "", false},
		{"cd a && make", "", false},
		{"cd a;ls", "", false},
		{"cd $HOME", "", false},
		{`cd "$HOME"`, "", false},
		{"cd *.d", "", false},
		{"cd -P /tmp", "", false},
		{`cd "unclosed`, "", false},
		{`cd trailing\`, "", false},
		{"cdx", "", false},
		{"ls", "", false},
	}
	for _, tt := range tests {
		dir, ok := ParseCd(tt.line)
		if dir != tt.dir || ok != tt.ok {
			t.Errorf("ParseCd(%q) = %q, %v; want %q, %v", tt.line, dir, ok, tt.dir, tt.ok)
		}
	}
}

func TestQuote(t *testing.T) {
	tests := []struct{ in, want string }{
		{"file.txt", "file.txt"},
		{"a-b_c+d@e%f:g,h=i/j", "a-b_c+d@e%f:g,h=i/j"},
		{"Отчёт.pdf", "Отчёт.pdf"},
		{"my file.txt", "'my file.txt'"},
		{"it's", `'it'\''s'`},
		{"a&b", "'a&b'"},
		{"~x", "'~x'"},
		{"", "''"},
	}
	for _, tt := range tests {
		if got := Quote(tt.in); got != tt.want {
			t.Errorf("Quote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRunnable(t *testing.T) {
	dir := t.TempDir()
	write := func(name, data string, mode os.FileMode) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(data), mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	script := write("script", "#!/bin/sh\necho hi\n", 0o755)
	if err := os.Symlink(script, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable() // the test binary is Mach-O
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path string
		want bool
	}{
		{script, true},
		{filepath.Join(dir, "link"), true},
		{write("short", "#!", 0o755), true},
		{self, true},
		{write("noexec.sh", "#!/bin/sh\n", 0o644), false},
		{write("photo.jpg", "\xff\xd8\xff\xe0JFIF", 0o755), false},
		{write("empty", "", 0o755), false},
		{dir, false},
		{filepath.Join(dir, "missing"), false},
	}
	for _, tt := range tests {
		if got := Runnable(tt.path); got != tt.want {
			t.Errorf("Runnable(%s) = %v, want %v", filepath.Base(tt.path), got, tt.want)
		}
	}
}
