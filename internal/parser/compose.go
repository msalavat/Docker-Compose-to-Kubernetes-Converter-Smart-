package parser

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseComposeFile reads and parses a docker-compose.yml file from the given path.
func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading compose file %q: %w", path, err)
	}
	return ParseComposeBytes(data)
}

// ParseComposeBytes parses docker-compose YAML from raw bytes.
func ParseComposeBytes(data []byte) (*ComposeFile, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty compose file")
	}

	// Substitute environment variables before parsing
	data = substituteEnvVars(data)

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Validate version
	if err := validateVersion(compose.Version); err != nil {
		return nil, err
	}

	if len(compose.Services) == 0 {
		return nil, fmt.Errorf("no services defined in compose file")
	}

	return &compose, nil
}

// validateVersion checks that the compose version is >= 3.8.
func validateVersion(version string) error {
	if version == "" {
		// Compose spec without version field is valid (v3+ implied)
		return nil
	}

	v := strings.TrimPrefix(version, "\"")
	v = strings.TrimSuffix(v, "\"")

	parts := strings.SplitN(v, ".", 2)
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid compose version %q: %w", version, err)
	}

	if major < 3 {
		return fmt.Errorf("unsupported compose version %q: minimum supported is 3.8", version)
	}

	if major == 3 && len(parts) > 1 {
		minor, err := strconv.Atoi(parts[1])
		if err == nil && minor < 8 {
			return fmt.Errorf("unsupported compose version %q: minimum supported is 3.8", version)
		}
	}

	return nil
}

// envVarRegex matches ${VAR}, ${VAR:-default}, ${VAR-default}
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// substituteEnvVars replaces ${VAR:-default} patterns with env values or defaults.
func substituteEnvVars(data []byte) []byte {
	return envVarRegex.ReplaceAllFunc(data, func(match []byte) []byte {
		// Remove ${ and }
		inner := string(match[2 : len(match)-1])

		// Check for default value separator
		var name, defaultVal string
		var hasDefault bool

		if idx := strings.Index(inner, ":-"); idx != -1 {
			name = inner[:idx]
			defaultVal = inner[idx+2:]
			hasDefault = true
		} else if idx := strings.Index(inner, "-"); idx != -1 {
			name = inner[:idx]
			defaultVal = inner[idx+1:]
			hasDefault = true
		} else {
			name = inner
		}

		if val, ok := os.LookupEnv(name); ok {
			return []byte(val)
		}
		if hasDefault {
			return []byte(defaultVal)
		}
		return match // Leave as-is if no env var and no default
	})
}
