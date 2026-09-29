package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	c := Config{
		Left:       Panel{Path: "/tmp", Mode: "full", Sort: "size"},
		Right:      Panel{Path: "/Users", Mode: "brief", Sort: "name"},
		ShowHidden: true,
		History:    []string{"ls", "make test"},
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	if got := Load(path); !reflect.DeepEqual(got, c) {
		t.Fatalf("got %+v, want %+v", got, c)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode())
	}
	if es, _ := os.ReadDir(filepath.Dir(path)); len(es) != 1 {
		t.Fatalf("left files behind: %v", es)
	}
}

func TestLoadBroken(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string]string{
		"empty":    "",
		"garbage":  "{not json",
		"wrongtyp": `{"left": 5, "history": "ls"}`,
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := Load(path); !reflect.DeepEqual(got, Config{}) {
			t.Errorf("%s: got %+v", name, got)
		}
	}
	if got := Load(filepath.Join(dir, "missing")); !reflect.DeepEqual(got, Config{}) {
		t.Errorf("missing: got %+v", got)
	}
	if got := Load(""); !reflect.DeepEqual(got, Config{}) {
		t.Errorf("no path: got %+v", got)
	}
}

func TestPath(t *testing.T) {
	t.Setenv("HOME", "/Users/x")
	if got := Path(); got != "/Users/x/.config/terminal-commander/config.json" {
		t.Fatalf("got %q", got)
	}
}
