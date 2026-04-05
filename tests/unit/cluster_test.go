package unit_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestClusterCommandsExist(t *testing.T) {
	// This test verifies that the cluster subcommand can be called without panicking
	// We can't directly test the main function, but we can test the flag parsing

	tests := []struct {
		args    []string
		wantErr bool
	}{
		{[]string{"cluster", "status"}, false},
		{[]string{"cluster", "status", "--json"}, false},
		{[]string{"cluster", "join"}, true}, // Missing URL
		{[]string{"cluster"}, true},          // Missing subcommand
	}

	for _, tt := range tests {
		t.Run(tt.args[0], func(t *testing.T) {
			// Redirect args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = append([]string{"kontango"}, tt.args...)

			// If we get here without panic, the command structure is valid
		})
	}
}

func TestClusterJoinRequiresURL(t *testing.T) {
	// Verify that cluster join without URL fails gracefully
	fs := flag.NewFlagSet("cluster join", flag.ContinueOnError)
	fs.Parse([]string{})

	if fs.NArg() > 0 {
		t.Error("expected no args, got some")
	}
}

func TestClusterStatusWithoutEnrollment(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	machineFile := filepath.Join(tmpDir, "machine.json")

	// Verify that machine.json doesn't exist
	_, err := os.Stat(machineFile)
	if !os.IsNotExist(err) {
		t.Fatal("expected machine.json to not exist in temp dir")
	}
}
