// Package config loads and saves the setup: panel directories, modes and
// sorting, hidden files and the command history.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Panel struct {
	Path string `json:"path"`
	Mode string `json:"mode"` // "brief" or "full"
	Sort string `json:"sort"` // "name", "extension", "time", "size" or "unsorted"
}

type Config struct {
	Left       Panel    `json:"left"`
	Right      Panel    `json:"right"`
	ShowHidden bool     `json:"showHidden"`
	History    []string `json:"history"`
}

// Path is ~/.config/terminal-commander/config.json, or "" without a home
// directory.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "terminal-commander", "config.json")
}

// Load reads the config at path. A missing or broken file gives the
// defaults (the zero Config), without an error.
func Load(path string) Config {
	var c Config
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &c) != nil {
		return Config{}
	}
	return c
}

// Save writes c to path, creating the directory. The file is replaced
// whole, and only its owner can read it: it has the command history.
func Save(path string, c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) // fails harmlessly after the rename
	_, err = f.Write(append(data, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
