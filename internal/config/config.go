// Package config loads and validates meshery-mcp-server configuration.
//
// Precedence, lowest to highest: defaults < config file < environment variables.
// The Meshery token is the exception: it is read from the environment only.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	keyMesheryURL = "meshery.url"
	keyLogLevel   = "log.level"

	envMesheryURL   = "MESHERY_SERVER_URL"
	envMesheryToken = "MESHERY_API_TOKEN" //nolint:gosec // G101: env var name, not a credential
	envLogLevel     = "LOG_LEVEL"

	defaultMesheryURL = "http://localhost:9081"
	defaultLogLevel   = LogLevelInfo
)

// fileTokenKeys are config file keys that must never hold a token.
// meshery.token was the documented location before the token became env-only.
var fileTokenKeys = []string{"meshery.token", "auth.token"}

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Config is the fully resolved, validated configuration.
type Config struct {
	Meshery MesheryConfig `mapstructure:"meshery"`
	Auth    AuthConfig    `mapstructure:"auth"`
	Log     LogConfig     `mapstructure:"log"`
}

// MesheryConfig describes where the Meshery server is. It holds no identity.
type MesheryConfig struct {
	URL string `mapstructure:"url"`
}

// RedactedURL returns the URL with any password replaced by "xxxxx", for logs.
func (m MesheryConfig) RedactedURL() string {
	u, err := url.Parse(m.URL)
	if err != nil {
		return "<unparsable URL>"
	}
	return u.Redacted()
}

// AuthConfig describes who the server calls Meshery as. It is kept apart
// from MesheryConfig so that where Meshery is and whose identity is used
// can change independently.
type AuthConfig struct {
	// Token is the Meshery provider token, read only from MESHERY_API_TOKEN.
	// Empty means requests carry no token, as with Meshery's Local provider.
	Token Secret `mapstructure:"-"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `mapstructure:"level"` // one of the LogLevel* constants
}

// Load builds the configuration. configPath is optional; an empty string
// means no file is read.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetDefault(keyMesheryURL, defaultMesheryURL)
	v.SetDefault(keyLogLevel, defaultLogLevel)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, ErrReadConfigFile(err, configPath)
		}
		for _, key := range fileTokenKeys {
			if v.IsSet(key) {
				return nil, ErrTokenInConfigFile(key)
			}
		}
	}

	// MustBindEnv only fails when called without a key, which would be a bug
	// here, not something a user can fix; panicking is appropriate.
	v.MustBindEnv(keyMesheryURL, envMesheryURL)
	v.MustBindEnv(keyLogLevel, envLogLevel)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, ErrDecodeConfig(err)
	}
	cfg.Auth.Token = Secret(os.Getenv(envMesheryToken))
	cfg.Log.Level = strings.ToLower(strings.TrimSpace(cfg.Log.Level))

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	u, err := url.Parse(c.Meshery.URL)
	if err != nil {
		return ErrInvalidMesheryURL("The URL could not be parsed")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidMesheryURL(fmt.Sprintf("Scheme of %v must be http or https", u.Redacted()))
	}
	if u.Host == "" {
		return ErrInvalidMesheryURL(fmt.Sprintf("URL %v has no host", u.Redacted()))
	}

	switch c.Log.Level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel(c.Log.Level)
	}
	return nil
}
