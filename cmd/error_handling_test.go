package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRunGetThreadReturnsErrorWhenUnauthenticated(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetConfigFile(filepath.Join(t.TempDir(), "config.toml"))

	SelectedProfile = "isolated-test"
	BackendURLOverride = "https://example.invalid"
	AppHostOverride = ""
	t.Cleanup(func() {
		SelectedProfile = ""
		BackendURLOverride = ""
		AppHostOverride = ""
	})

	err := runGetThread(nil, []string{"00000000-0000-4000-8000-000000000000"})
	if err == nil {
		t.Fatal("expected missing credentials to return an error")
	}
	if !strings.Contains(err.Error(), "missing credentials") {
		t.Fatalf("expected missing credentials error, got %v", err)
	}
}

func TestSaveProfileUsesOwnerOnlyConfigPermissions(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	viper.SetConfigFile(configPath)

	err := saveProfile(profileConfig{
		Name:        "test",
		BackendURL:  "https://example.invalid",
		AppHost:     "https://app.example.invalid",
		AccessToken: "access-token",
		Client:      "client",
		UID:         "user@example.invalid",
	}, true)
	if err != nil {
		t.Fatalf("saveProfile returned error: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected config file mode 0600, got %04o", got)
	}
}
