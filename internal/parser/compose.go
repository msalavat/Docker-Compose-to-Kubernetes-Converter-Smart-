// Package parser reads and normalizes Docker Compose v3.8+ files into structured Go types.
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

// ParseComposeFiles reads and merges multiple docker-compose files.
// Later files override values from earlier files (like docker-compose.override.yml).
func ParseComposeFiles(paths []string) (*ComposeFile, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no compose files specified")
	}

	base, err := ParseComposeFile(paths[0])
	if err != nil {
		return nil, fmt.Errorf("parsing %q: %w", paths[0], err)
	}

	for _, path := range paths[1:] {
		override, err := ParseComposeFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", path, err)
		}
		mergeCompose(base, override)
	}

	return base, nil
}

// mergeCompose merges override into base. Override values take precedence for non-zero fields.
func mergeCompose(base, override *ComposeFile) {
	if override.Version != "" {
		base.Version = override.Version
	}

	// Merge services
	if base.Services == nil {
		base.Services = make(map[string]ServiceConfig)
	}
	for name, overSvc := range override.Services {
		if baseSvc, exists := base.Services[name]; exists {
			base.Services[name] = mergeService(baseSvc, overSvc)
		} else {
			base.Services[name] = overSvc
		}
	}

	// Merge top-level volumes
	if override.Volumes != nil {
		if base.Volumes == nil {
			base.Volumes = make(map[string]VolumeConfig)
		}
		for k, v := range override.Volumes {
			base.Volumes[k] = v
		}
	}

	// Merge top-level networks
	if override.Networks != nil {
		if base.Networks == nil {
			base.Networks = make(map[string]NetworkConfig)
		}
		for k, v := range override.Networks {
			base.Networks[k] = v
		}
	}

	// Merge top-level secrets
	if override.Secrets != nil {
		if base.Secrets == nil {
			base.Secrets = make(map[string]SecretConfig)
		}
		for k, v := range override.Secrets {
			base.Secrets[k] = v
		}
	}

	// Merge top-level configs
	if override.Configs != nil {
		if base.Configs == nil {
			base.Configs = make(map[string]ConfigConfig)
		}
		for k, v := range override.Configs {
			base.Configs[k] = v
		}
	}
}

// mergeService merges an override service config into a base service config.
func mergeService(base, override ServiceConfig) ServiceConfig {
	if override.Image != "" {
		base.Image = override.Image
	}
	if override.ContainerName != "" {
		base.ContainerName = override.ContainerName
	}
	if override.Restart != "" {
		base.Restart = override.Restart
	}
	if override.User != "" {
		base.User = override.User
	}
	if override.WorkingDir != "" {
		base.WorkingDir = override.WorkingDir
	}

	// Append ports
	if len(override.Ports) > 0 {
		base.Ports = append(base.Ports, override.Ports...)
	}

	// Append volumes
	if len(override.Volumes) > 0 {
		base.Volumes = append(base.Volumes, override.Volumes...)
	}

	// Merge environment (override wins on key conflicts)
	if len(override.Environment) > 0 {
		if base.Environment == nil {
			base.Environment = make(map[string]string)
		}
		for k, v := range override.Environment {
			base.Environment[k] = v
		}
	}

	// Override scalar/struct fields if set
	if len(override.EnvFile) > 0 {
		base.EnvFile = override.EnvFile
	}
	if len(override.Command) > 0 {
		base.Command = override.Command
	}
	if len(override.Entrypoint) > 0 {
		base.Entrypoint = override.Entrypoint
	}
	if override.Healthcheck != nil {
		base.Healthcheck = override.Healthcheck
	}
	if override.Deploy != nil {
		base.Deploy = override.Deploy
	}
	if len(override.Labels) > 0 {
		base.Labels = override.Labels
	}
	if len(override.Networks) > 0 {
		base.Networks = override.Networks
	}
	if len(override.DependsOn) > 0 {
		base.DependsOn = override.DependsOn
	}
	if len(override.Expose) > 0 {
		base.Expose = override.Expose
	}
	if len(override.CapAdd) > 0 {
		base.CapAdd = override.CapAdd
	}
	if len(override.CapDrop) > 0 {
		base.CapDrop = override.CapDrop
	}
	if len(override.SecurityOpt) > 0 {
		base.SecurityOpt = override.SecurityOpt
	}
	if len(override.ExtraHosts) > 0 {
		base.ExtraHosts = override.ExtraHosts
	}
	if len(override.DNS) > 0 {
		base.DNS = override.DNS
	}
	if len(override.DNSSearch) > 0 {
		base.DNSSearch = override.DNSSearch
	}
	if override.Logging != nil {
		base.Logging = override.Logging
	}
	if override.Build != nil {
		base.Build = override.Build
	}

	// Bool fields: only override if true (can't distinguish false from unset with plain bool)
	if override.StdinOpen {
		base.StdinOpen = true
	}
	if override.Tty {
		base.Tty = true
	}
	if override.Privileged {
		base.Privileged = true
	}
	if override.ReadOnly {
		base.ReadOnly = true
	}

	return base
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
