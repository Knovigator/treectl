package main

import (
	"strings"
	"testing"

	treectlcmd "github.com/Knovigator/treectl/cmd"
)

func TestRootExecuteReturnsCommandErrors(t *testing.T) {
	treectlcmd.SelectedProfile = "isolated-test"
	treectlcmd.BackendURLOverride = "https://example.invalid"
	rootCmd.SetArgs([]string{"get", "thread", "00000000-0000-4000-8000-000000000000"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		treectlcmd.SelectedProfile = ""
		treectlcmd.BackendURLOverride = ""
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected command failure to return an error")
	}
	if !strings.Contains(err.Error(), "missing credentials") {
		t.Fatalf("expected missing credentials error, got %v", err)
	}
}
