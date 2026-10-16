package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meshery/meshkit/errors"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	const fileYAML = "meshery:\n  url: https://file.example:9081\nlog:\n  level: warn\n"

	tests := []struct {
		name      string
		envURL    string
		envToken  string
		envLevel  string
		fileYAML  string // "" = no config file
		path      string // explicit path; overrides fileYAML
		wantURL   string
		wantToken Secret
		wantLevel string
		wantErr   string // expected short description; "" = expect success
	}{
		{name: "defaults only", wantURL: defaultMesheryURL, wantLevel: LogLevelInfo},
		{name: "env overrides default", envURL: "https://env.example", envToken: "env-token", envLevel: "debug",
			wantURL: "https://env.example", wantToken: "env-token", wantLevel: LogLevelDebug},
		{name: "file only", fileYAML: fileYAML,
			wantURL: "https://file.example:9081", wantLevel: LogLevelWarn},
		{name: "env beats file per key", fileYAML: fileYAML, envURL: "https://env.example", envToken: "env-token",
			wantURL: "https://env.example", wantToken: "env-token", wantLevel: LogLevelWarn},
		{name: "log level is case-insensitive", envLevel: " DEBUG ",
			wantURL: defaultMesheryURL, wantLevel: LogLevelDebug},
		{name: "repo config_example.yml loads", path: filepath.Join("..", "..", "config_example.yml"),
			wantURL: defaultMesheryURL, wantLevel: LogLevelInfo},
		{name: "invalid scheme", envURL: "ftp://x", wantErr: "Invalid Meshery server URL"},
		{name: "missing scheme", envURL: "localhost:9081", wantErr: "Invalid Meshery server URL"},
		{name: "unparsable URL", envURL: "http://u:p@host:90x81", wantErr: "Invalid Meshery server URL"},
		{name: "invalid log level", envLevel: "verbose", wantErr: "Invalid log level"},
		{name: "missing file", path: filepath.Join(t.TempDir(), "nope.yaml"), wantErr: "Unable to read the config file"},

		// Token
		{name: "token from the environment", envToken: "env-token",
			wantURL: defaultMesheryURL, wantToken: "env-token", wantLevel: LogLevelInfo},
		{name: "token under meshery in file", fileYAML: "meshery:\n  token: file-token\n",
			wantErr: "Token is not allowed in the config file"},
		{name: "token under auth in file", fileYAML: "auth:\n  token: file-token\n",
			wantErr: "Token is not allowed in the config file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always set all: an empty value counts as unset.
			t.Setenv(envMesheryURL, tt.envURL)
			t.Setenv(envMesheryToken, tt.envToken)
			t.Setenv(envLogLevel, tt.envLevel)

			path := tt.path
			if path == "" && tt.fileYAML != "" {
				path = writeConfigFile(t, tt.fileYAML)
			}

			cfg, err := Load(path)
			if tt.wantErr != "" {
				if err == nil || errors.GetSDescription(err) != tt.wantErr {
					t.Fatalf("want error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Meshery.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", cfg.Meshery.URL, tt.wantURL)
			}
			if cfg.Auth.Token != tt.wantToken {
				t.Errorf("Auth.Token = %q, want %q", string(cfg.Auth.Token), string(tt.wantToken))
			}
			if cfg.Log.Level != tt.wantLevel {
				t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, tt.wantLevel)
			}
		})
	}
}

func TestInvalidURLErrorDoesNotLeakCredentials(t *testing.T) {
	t.Setenv(envMesheryToken, "")
	t.Setenv(envLogLevel, "")
	for _, raw := range []string{"http://admin:s3cret@host:90x81", "ftp://admin:s3cret@host"} {
		t.Setenv(envMesheryURL, raw)
		_, err := Load("")
		if err == nil {
			t.Fatalf("%s: expected an error", raw)
		}
		if strings.Contains(err.Error(), "s3cret") {
			t.Errorf("password leaked into error: %v", err)
		}
	}
}

func TestTokenErrorsDoNotLeakToken(t *testing.T) {
	t.Setenv(envMesheryURL, "")
	t.Setenv(envLogLevel, "")
	t.Setenv(envMesheryToken, "super-secret")

	_, err := Load(writeConfigFile(t, "meshery:\n  token: file-secret\n"))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, secret := range []string{"super-secret", "file-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("token leaked into error: %v", err)
		}
	}
}

func TestRedactedURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{url: "http://localhost:9081", want: "http://localhost:9081"},
		{url: "https://admin:s3cret@meshery.example", want: "https://admin:xxxxx@meshery.example"},
		{url: "http://u:p@host:90x81", want: "<unparsable URL>"},
	}
	for _, tt := range tests {
		got := MesheryConfig{URL: tt.url}.RedactedURL()
		if got != tt.want {
			t.Errorf("RedactedURL(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestSecretIsRedacted(t *testing.T) {
	cfg := Config{
		Meshery: MesheryConfig{URL: "http://x"},
		Auth:    AuthConfig{Token: "super-secret"},
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, out := range []string{fmt.Sprint(cfg), fmt.Sprintf("%+v", cfg), string(b)} {
		if strings.Contains(out, "super-secret") {
			t.Errorf("token leaked: %s", out)
		}
	}
}
