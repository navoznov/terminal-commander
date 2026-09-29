package app

import (
	"os"
	"testing"
)

// TestMain keeps tests from opening documents in real apps.
func TestMain(m *testing.M) {
	openCmd = "/usr/bin/true"
	os.Exit(m.Run())
}
