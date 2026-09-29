package main

import (
	"testing"

	"github.com/navoznov/terminal-commander/internal/config"
)

func TestStartDirs(t *testing.T) {
	saved := config.Config{Left: config.Panel{Path: "/l"}, Right: config.Panel{Path: "/r"}}
	tests := []struct {
		cfg         config.Config
		args        []string
		left, right string
	}{
		{config.Config{}, nil, "/cwd", "/cwd"},
		{saved, nil, "/l", "/r"},
		{saved, []string{"/x"}, "/x", "/r"},
		{config.Config{}, []string{"/x/../y"}, "/y", "/cwd"},
	}
	for _, tt := range tests {
		l, r := startDirs(tt.cfg, tt.args, "/cwd")
		if l != tt.left || r != tt.right {
			t.Errorf("startDirs(%+v, %q) = %q, %q; want %q, %q", tt.cfg, tt.args, l, r, tt.left, tt.right)
		}
	}
}
