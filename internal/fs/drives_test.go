package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDrives(t *testing.T) {
	vol := t.TempDir()
	for _, name := range []string{"USB", "Backup", ".timemachine"} {
		if err := os.Mkdir(filepath.Join(vol, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("/", filepath.Join(vol, "Macintosh HD")); err != nil {
		t.Fatal(err)
	}
	got := drives(vol, "/Users/nc")
	want := []Drive{
		{"/", "/"},
		{"~", "/Users/nc"},
		{"Backup", filepath.Join(vol, "Backup")},
		{"USB", filepath.Join(vol, "USB")},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestDrivesWithoutVolumes(t *testing.T) {
	got := drives(filepath.Join(t.TempDir(), "missing"), "")
	if len(got) != 1 || got[0].Path != "/" {
		t.Fatalf("got %v", got)
	}
}
