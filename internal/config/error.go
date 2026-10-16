package config

import (
	"fmt"

	"github.com/meshery/meshkit/errors"
)

const (
	ErrReadConfigFileCode    = "meshery-mcp-server-1010"
	ErrDecodeConfigCode      = "meshery-mcp-server-1011"
	ErrInvalidMesheryURLCode = "meshery-mcp-server-1012"
	ErrInvalidLogLevelCode   = "meshery-mcp-server-1013"
	ErrTokenInConfigFileCode = "meshery-mcp-server-1014" //nolint:gosec // G101: error code, not a credential
)

// ErrReadConfigFile is returned when the --config file cannot be read or parsed.
func ErrReadConfigFile(err error, path string) error {
	return errors.New(ErrReadConfigFileCode, errors.Alert,
		[]string{"Unable to read the config file"},
		[]string{fmt.Sprintf("Reading %q failed: %v", path, err)},
		[]string{"The file passed with --config does not exist, is not readable, or is not valid YAML"},
		[]string{"Check the path given to --config and that the file contains valid YAML"})
}

// ErrDecodeConfig is returned when config values do not match the expected types.
func ErrDecodeConfig(err error) error {
	return errors.New(ErrDecodeConfigCode, errors.Alert,
		[]string{"Unable to decode configuration"},
		[]string{err.Error()},
		[]string{"A setting in the config file has the wrong type"},
		[]string{"Check that every setting in the config file is a plain string"})
}

// ErrInvalidMesheryURL is returned when MESHERY_SERVER_URL / meshery.url is unusable.
// reason must never contain the raw URL, which may embed credentials.
func ErrInvalidMesheryURL(reason string) error {
	return errors.New(ErrInvalidMesheryURLCode, errors.Alert,
		[]string{"Invalid Meshery server URL"},
		[]string{reason},
		[]string{"MESHERY_SERVER_URL or meshery.url is malformed or uses an unsupported scheme"},
		[]string{"Set MESHERY_SERVER_URL to a URL like http://localhost:9081"})
}

// ErrInvalidLogLevel is returned when LOG_LEVEL / log.level is not a supported level.
func ErrInvalidLogLevel(level string) error {
	return errors.New(ErrInvalidLogLevelCode, errors.Alert,
		[]string{"Invalid log level"},
		[]string{fmt.Sprintf("Log level %q is not supported", level)},
		[]string{"LOG_LEVEL or log.level has a typo"},
		[]string{"Set LOG_LEVEL to one of: debug, info, warn, error"})
}

// ErrTokenInConfigFile is returned when the config file contains a token.
// The token is read from MESHERY_API_TOKEN only.
func ErrTokenInConfigFile(key string) error {
	return errors.New(ErrTokenInConfigFileCode, errors.Alert,
		[]string{"Token is not allowed in the config file"},
		[]string{fmt.Sprintf("The config file sets %q", key)},
		[]string{"The token was put in the config file, which is no longer supported"},
		[]string{"Remove the token from the config file and set MESHERY_API_TOKEN instead"})
}
