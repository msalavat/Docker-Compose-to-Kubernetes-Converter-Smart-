package parser

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a parsed docker-compose.yml file.
type ComposeFile struct {
	Version  string                   `yaml:"version"`
	Services map[string]ServiceConfig `yaml:"services"`
	Volumes  map[string]VolumeConfig  `yaml:"volumes,omitempty"`
	Networks map[string]NetworkConfig `yaml:"networks,omitempty"`
	Secrets  map[string]SecretConfig  `yaml:"secrets,omitempty"`
	Configs  map[string]ConfigConfig  `yaml:"configs,omitempty"`
}

// ServiceConfig represents a single service in docker-compose.
type ServiceConfig struct {
	Image         string             `yaml:"image,omitempty"`
	ContainerName string             `yaml:"container_name,omitempty"`
	Ports         []PortConfig       `yaml:"-"`
	Volumes       []VolumeMount      `yaml:"-"`
	Environment   map[string]string  `yaml:"-"`
	EnvFile       StringList         `yaml:"env_file,omitempty"`
	DependsOn     DependsOn          `yaml:"-"`
	Healthcheck   *HealthcheckConfig `yaml:"healthcheck,omitempty"`
	Deploy        *DeployConfig      `yaml:"deploy,omitempty"`
	Command       ShellCommand       `yaml:"-"`
	Entrypoint    ShellCommand       `yaml:"-"`
	Labels        Labels             `yaml:"-"`
	Networks      ServiceNetworks    `yaml:"-"`
	Expose        []string           `yaml:"expose,omitempty"`
	Restart       string             `yaml:"restart,omitempty"`
	User          string             `yaml:"user,omitempty"`
	WorkingDir    string             `yaml:"working_dir,omitempty"`
	StdinOpen     bool               `yaml:"stdin_open,omitempty"`
	Tty           bool               `yaml:"tty,omitempty"`
	Privileged    bool               `yaml:"privileged,omitempty"`
	ReadOnly      bool               `yaml:"read_only,omitempty"`
	CapAdd        []string           `yaml:"cap_add,omitempty"`
	CapDrop       []string           `yaml:"cap_drop,omitempty"`
	SecurityOpt   []string           `yaml:"security_opt,omitempty"`
	Sysctls       Sysctls            `yaml:"-"`
	ExtraHosts    []string           `yaml:"extra_hosts,omitempty"`
	DNS           StringList         `yaml:"-"`
	DNSSearch     StringList         `yaml:"-"`
	Logging       *LoggingConfig     `yaml:"logging,omitempty"`
	Build         *BuildConfig       `yaml:"-"`
}

// serviceConfigRaw is used internally to unmarshal fields that need custom handling.
type serviceConfigRaw struct {
	Image         string             `yaml:"image,omitempty"`
	ContainerName string             `yaml:"container_name,omitempty"`
	Ports         yaml.Node          `yaml:"ports,omitempty"`
	Volumes       yaml.Node          `yaml:"volumes,omitempty"`
	Environment   yaml.Node          `yaml:"environment,omitempty"`
	EnvFile       StringList         `yaml:"env_file,omitempty"`
	DependsOn     yaml.Node          `yaml:"depends_on,omitempty"`
	Healthcheck   *HealthcheckConfig `yaml:"healthcheck,omitempty"`
	Deploy        *DeployConfig      `yaml:"deploy,omitempty"`
	Command       yaml.Node          `yaml:"command,omitempty"`
	Entrypoint    yaml.Node          `yaml:"entrypoint,omitempty"`
	Labels        yaml.Node          `yaml:"labels,omitempty"`
	Networks      yaml.Node          `yaml:"networks,omitempty"`
	Expose        []string           `yaml:"expose,omitempty"`
	Restart       string             `yaml:"restart,omitempty"`
	User          string             `yaml:"user,omitempty"`
	WorkingDir    string             `yaml:"working_dir,omitempty"`
	StdinOpen     bool               `yaml:"stdin_open,omitempty"`
	Tty           bool               `yaml:"tty,omitempty"`
	Privileged    bool               `yaml:"privileged,omitempty"`
	ReadOnly      bool               `yaml:"read_only,omitempty"`
	CapAdd        []string           `yaml:"cap_add,omitempty"`
	CapDrop       []string           `yaml:"cap_drop,omitempty"`
	SecurityOpt   []string           `yaml:"security_opt,omitempty"`
	Sysctls       yaml.Node          `yaml:"sysctls,omitempty"`
	ExtraHosts    []string           `yaml:"extra_hosts,omitempty"`
	DNS           yaml.Node          `yaml:"dns,omitempty"`
	DNSSearch     yaml.Node          `yaml:"dns_search,omitempty"`
	Logging       *LoggingConfig     `yaml:"logging,omitempty"`
	Build         yaml.Node          `yaml:"build,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for ServiceConfig.
func (s *ServiceConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw serviceConfigRaw
	if err := value.Decode(&raw); err != nil {
		return err
	}

	// Copy simple fields
	s.Image = raw.Image
	s.ContainerName = raw.ContainerName
	s.EnvFile = raw.EnvFile
	s.Healthcheck = raw.Healthcheck
	s.Deploy = raw.Deploy
	s.Expose = raw.Expose
	s.Restart = raw.Restart
	s.User = raw.User
	s.WorkingDir = raw.WorkingDir
	s.StdinOpen = raw.StdinOpen
	s.Tty = raw.Tty
	s.Privileged = raw.Privileged
	s.ReadOnly = raw.ReadOnly
	s.CapAdd = raw.CapAdd
	s.CapDrop = raw.CapDrop
	s.SecurityOpt = raw.SecurityOpt
	s.ExtraHosts = raw.ExtraHosts
	s.Logging = raw.Logging

	// Parse ports
	if raw.Ports.Kind != 0 {
		ports, err := parsePorts(&raw.Ports)
		if err != nil {
			return fmt.Errorf("parsing ports: %w", err)
		}
		s.Ports = ports
	}

	// Parse volumes
	if raw.Volumes.Kind != 0 {
		vols, err := parseVolumes(&raw.Volumes)
		if err != nil {
			return fmt.Errorf("parsing volumes: %w", err)
		}
		s.Volumes = vols
	}

	// Parse environment
	if raw.Environment.Kind != 0 {
		env, err := parseEnvironment(&raw.Environment)
		if err != nil {
			return fmt.Errorf("parsing environment: %w", err)
		}
		s.Environment = env
	}

	// Parse depends_on
	if raw.DependsOn.Kind != 0 {
		deps, err := parseDependsOn(&raw.DependsOn)
		if err != nil {
			return fmt.Errorf("parsing depends_on: %w", err)
		}
		s.DependsOn = deps
	}

	// Parse command
	if raw.Command.Kind != 0 {
		cmd, err := parseShellCommand(&raw.Command)
		if err != nil {
			return fmt.Errorf("parsing command: %w", err)
		}
		s.Command = cmd
	}

	// Parse entrypoint
	if raw.Entrypoint.Kind != 0 {
		ep, err := parseShellCommand(&raw.Entrypoint)
		if err != nil {
			return fmt.Errorf("parsing entrypoint: %w", err)
		}
		s.Entrypoint = ep
	}

	// Parse labels
	if raw.Labels.Kind != 0 {
		labels, err := parseLabels(&raw.Labels)
		if err != nil {
			return fmt.Errorf("parsing labels: %w", err)
		}
		s.Labels = labels
	}

	// Parse networks
	if raw.Networks.Kind != 0 {
		nets, err := parseServiceNetworks(&raw.Networks)
		if err != nil {
			return fmt.Errorf("parsing networks: %w", err)
		}
		s.Networks = nets
	}

	// Parse sysctls
	if raw.Sysctls.Kind != 0 {
		sc, err := parseSysctls(&raw.Sysctls)
		if err != nil {
			return fmt.Errorf("parsing sysctls: %w", err)
		}
		s.Sysctls = sc
	}

	// Parse dns
	if raw.DNS.Kind != 0 {
		dns, err := parseStringList(&raw.DNS)
		if err != nil {
			return fmt.Errorf("parsing dns: %w", err)
		}
		s.DNS = dns
	}

	// Parse dns_search
	if raw.DNSSearch.Kind != 0 {
		ds, err := parseStringList(&raw.DNSSearch)
		if err != nil {
			return fmt.Errorf("parsing dns_search: %w", err)
		}
		s.DNSSearch = ds
	}

	// Parse build
	if raw.Build.Kind != 0 {
		b, err := parseBuild(&raw.Build)
		if err != nil {
			return fmt.Errorf("parsing build: %w", err)
		}
		s.Build = b
	}

	return nil
}

// PortConfig represents a normalized port mapping.
type PortConfig struct {
	HostPort      uint32
	ContainerPort uint32
	Protocol      string // "tcp" or "udp"
}

// VolumeMount represents a normalized volume mount.
type VolumeMount struct {
	Type     string // "volume", "bind", "tmpfs"
	Source   string
	Target   string
	ReadOnly bool
}

// DependsOn represents service dependencies.
type DependsOn map[string]DependsOnCondition

// DependsOnCondition represents a dependency condition.
type DependsOnCondition struct {
	Condition string `yaml:"condition,omitempty"`
}

// HealthcheckConfig represents a service healthcheck.
type HealthcheckConfig struct {
	Test        ShellCommand `yaml:"-"`
	Interval    string       `yaml:"interval,omitempty"`
	Timeout     string       `yaml:"timeout,omitempty"`
	Retries     int          `yaml:"retries,omitempty"`
	StartPeriod string       `yaml:"start_period,omitempty"`
	Disable     bool         `yaml:"disable,omitempty"`
}

// healthcheckRaw is used for custom unmarshaling of HealthcheckConfig.
type healthcheckRaw struct {
	Test        yaml.Node `yaml:"test,omitempty"`
	Interval    string    `yaml:"interval,omitempty"`
	Timeout     string    `yaml:"timeout,omitempty"`
	Retries     int       `yaml:"retries,omitempty"`
	StartPeriod string    `yaml:"start_period,omitempty"`
	Disable     bool      `yaml:"disable,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for HealthcheckConfig.
func (h *HealthcheckConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw healthcheckRaw
	if err := value.Decode(&raw); err != nil {
		return err
	}
	h.Interval = raw.Interval
	h.Timeout = raw.Timeout
	h.Retries = raw.Retries
	h.StartPeriod = raw.StartPeriod
	h.Disable = raw.Disable

	if raw.Test.Kind != 0 {
		cmd, err := parseShellCommand(&raw.Test)
		if err != nil {
			return fmt.Errorf("parsing healthcheck test: %w", err)
		}
		h.Test = cmd
	}
	return nil
}

// DeployConfig represents deploy configuration.
type DeployConfig struct {
	Replicas      *int           `yaml:"replicas,omitempty"`
	Resources     *Resources     `yaml:"resources,omitempty"`
	RestartPolicy *RestartPolicy `yaml:"restart_policy,omitempty"`
	Placement     *Placement     `yaml:"placement,omitempty"`
	Labels        Labels         `yaml:"-"`
}

// deployConfigRaw for custom unmarshaling.
type deployConfigRaw struct {
	Replicas      *int           `yaml:"replicas,omitempty"`
	Resources     *Resources     `yaml:"resources,omitempty"`
	RestartPolicy *RestartPolicy `yaml:"restart_policy,omitempty"`
	Placement     *Placement     `yaml:"placement,omitempty"`
	Labels        yaml.Node      `yaml:"labels,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for DeployConfig.
func (d *DeployConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw deployConfigRaw
	if err := value.Decode(&raw); err != nil {
		return err
	}
	d.Replicas = raw.Replicas
	d.Resources = raw.Resources
	d.RestartPolicy = raw.RestartPolicy
	d.Placement = raw.Placement
	if raw.Labels.Kind != 0 {
		labels, err := parseLabels(&raw.Labels)
		if err != nil {
			return fmt.Errorf("parsing deploy labels: %w", err)
		}
		d.Labels = labels
	}
	return nil
}

// Resources represents resource constraints.
type Resources struct {
	Limits       *ResourceSpec `yaml:"limits,omitempty"`
	Reservations *ResourceSpec `yaml:"reservations,omitempty"`
}

// ResourceSpec represents a resource specification (CPU/memory).
type ResourceSpec struct {
	CPUs   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// RestartPolicy represents a restart policy.
type RestartPolicy struct {
	Condition   string `yaml:"condition,omitempty"`
	Delay       string `yaml:"delay,omitempty"`
	MaxAttempts int    `yaml:"max_attempts,omitempty"`
	Window      string `yaml:"window,omitempty"`
}

// Placement represents deployment placement constraints.
type Placement struct {
	Constraints []string `yaml:"constraints,omitempty"`
}

// VolumeConfig represents a top-level volume definition.
type VolumeConfig struct {
	Driver     string            `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	External   bool              `yaml:"external,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	Name       string            `yaml:"name,omitempty"`
}

// NetworkConfig represents a top-level network definition.
type NetworkConfig struct {
	Driver     string            `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	External   bool              `yaml:"external,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
	Name       string            `yaml:"name,omitempty"`
}

// SecretConfig represents a top-level secret definition.
type SecretConfig struct {
	File     string `yaml:"file,omitempty"`
	External bool   `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// ConfigConfig represents a top-level config definition.
type ConfigConfig struct {
	File     string `yaml:"file,omitempty"`
	External bool   `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// LoggingConfig represents logging configuration.
type LoggingConfig struct {
	Driver  string            `yaml:"driver,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
}

// BuildConfig represents a build configuration.
type BuildConfig struct {
	Context    string `yaml:"context,omitempty"`
	Dockerfile string `yaml:"dockerfile,omitempty"`
}

// ShellCommand can be a single string or a list of strings.
type ShellCommand []string

// StringList can be a single string or a list of strings.
type StringList []string

// UnmarshalYAML implements custom unmarshaling for StringList.
func (s *StringList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*s = []string{value.Value}
		return nil
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*s = list
		return nil
	default:
		return fmt.Errorf("expected string or list, got %v", value.Kind)
	}
}

// Labels can be a list of "key=value" strings or a map.
type Labels map[string]string

// Sysctls can be a list of "key=value" strings or a map.
type Sysctls map[string]string

// ServiceNetworks can be a list of network names or a map of network configs.
type ServiceNetworks map[string]*ServiceNetworkConfig

// ServiceNetworkConfig represents per-service network configuration.
type ServiceNetworkConfig struct {
	Aliases []string `yaml:"aliases,omitempty"`
}

// --- Parser helpers ---

func parsePorts(node *yaml.Node) ([]PortConfig, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected sequence for ports")
	}

	var ports []PortConfig
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			// Short syntax: "8080:80", "80", "8080:80/udp"
			p, err := parsePortString(item.Value)
			if err != nil {
				return nil, err
			}
			ports = append(ports, p)
		case yaml.MappingNode:
			// Long syntax: {target: 80, published: 8080, protocol: tcp}
			var long struct {
				Target    uint32 `yaml:"target"`
				Published uint32 `yaml:"published"`
				Protocol  string `yaml:"protocol"`
			}
			if err := item.Decode(&long); err != nil {
				return nil, fmt.Errorf("parsing long port syntax: %w", err)
			}
			proto := long.Protocol
			if proto == "" {
				proto = "tcp"
			}
			ports = append(ports, PortConfig{
				HostPort:      long.Published,
				ContainerPort: long.Target,
				Protocol:      proto,
			})
		default:
			return nil, fmt.Errorf("unexpected port entry type")
		}
	}
	return ports, nil
}

func parsePortString(s string) (PortConfig, error) {
	proto := "tcp"
	if idx := strings.Index(s, "/"); idx != -1 {
		proto = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		port, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return PortConfig{}, fmt.Errorf("invalid port %q: %w", s, err)
		}
		return PortConfig{ContainerPort: uint32(port), Protocol: proto}, nil
	case 2:
		host, err := strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return PortConfig{}, fmt.Errorf("invalid host port %q: %w", parts[0], err)
		}
		container, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return PortConfig{}, fmt.Errorf("invalid container port %q: %w", parts[1], err)
		}
		return PortConfig{HostPort: uint32(host), ContainerPort: uint32(container), Protocol: proto}, nil
	case 3:
		// ip:hostPort:containerPort — ignore IP
		host, err := strconv.ParseUint(parts[1], 10, 32)
		if err != nil {
			return PortConfig{}, fmt.Errorf("invalid host port %q: %w", parts[1], err)
		}
		container, err := strconv.ParseUint(parts[2], 10, 32)
		if err != nil {
			return PortConfig{}, fmt.Errorf("invalid container port %q: %w", parts[2], err)
		}
		return PortConfig{HostPort: uint32(host), ContainerPort: uint32(container), Protocol: proto}, nil
	default:
		return PortConfig{}, fmt.Errorf("invalid port format %q", s)
	}
}

func parseVolumes(node *yaml.Node) ([]VolumeMount, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected sequence for volumes")
	}

	var vols []VolumeMount
	for _, item := range node.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			v, err := parseVolumeString(item.Value)
			if err != nil {
				return nil, err
			}
			vols = append(vols, v)
		case yaml.MappingNode:
			var long struct {
				Type     string `yaml:"type"`
				Source   string `yaml:"source"`
				Target   string `yaml:"target"`
				ReadOnly bool   `yaml:"read_only"`
			}
			if err := item.Decode(&long); err != nil {
				return nil, fmt.Errorf("parsing long volume syntax: %w", err)
			}
			volType := long.Type
			if volType == "" {
				volType = "volume"
			}
			vols = append(vols, VolumeMount{
				Type:     volType,
				Source:   long.Source,
				Target:   long.Target,
				ReadOnly: long.ReadOnly,
			})
		default:
			return nil, fmt.Errorf("unexpected volume entry type")
		}
	}
	return vols, nil
}

func parseVolumeString(s string) (VolumeMount, error) {
	readOnly := false
	if strings.HasSuffix(s, ":ro") {
		readOnly = true
		s = s[:len(s)-3]
	} else if strings.HasSuffix(s, ":rw") {
		s = s[:len(s)-3]
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 1 {
		// Anonymous volume: just a path
		return VolumeMount{Type: "volume", Target: parts[0]}, nil
	}

	source := parts[0]
	target := parts[1]

	volType := "volume"
	if strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") {
		volType = "bind"
	}

	return VolumeMount{
		Type:     volType,
		Source:   source,
		Target:   target,
		ReadOnly: readOnly,
	}, nil
}

func parseEnvironment(node *yaml.Node) (map[string]string, error) {
	env := make(map[string]string)

	switch node.Kind {
	case yaml.SequenceNode:
		// List format: ["KEY=value", "KEY2=value2"]
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("expected string in environment list")
			}
			k, v := parseEnvEntry(item.Value)
			env[k] = v
		}
	case yaml.MappingNode:
		// Map format: {KEY: value, KEY2: value2}
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i].Value
			val := node.Content[i+1].Value
			env[key] = val
		}
	default:
		return nil, fmt.Errorf("expected list or map for environment")
	}
	return env, nil
}

func parseEnvEntry(s string) (string, string) {
	idx := strings.IndexByte(s, '=')
	if idx == -1 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}

func parseDependsOn(node *yaml.Node) (DependsOn, error) {
	deps := make(DependsOn)

	switch node.Kind {
	case yaml.SequenceNode:
		// Simple list: [db, cache]
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("expected string in depends_on list")
			}
			deps[item.Value] = DependsOnCondition{Condition: "service_started"}
		}
	case yaml.MappingNode:
		// Extended format: {db: {condition: service_healthy}}
		for i := 0; i < len(node.Content)-1; i += 2 {
			name := node.Content[i].Value
			var cond DependsOnCondition
			if node.Content[i+1].Kind == yaml.MappingNode {
				if err := node.Content[i+1].Decode(&cond); err != nil {
					return nil, fmt.Errorf("parsing depends_on condition for %q: %w", name, err)
				}
			}
			if cond.Condition == "" {
				cond.Condition = "service_started"
			}
			deps[name] = cond
		}
	default:
		return nil, fmt.Errorf("expected list or map for depends_on")
	}
	return deps, nil
}

func parseShellCommand(node *yaml.Node) (ShellCommand, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return ShellCommand{node.Value}, nil
	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return nil, err
		}
		return ShellCommand(list), nil
	default:
		return nil, fmt.Errorf("expected string or list for command")
	}
}

func parseLabels(node *yaml.Node) (Labels, error) {
	labels := make(Labels)

	switch node.Kind {
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("expected string in labels list")
			}
			k, v := parseEnvEntry(item.Value)
			labels[k] = v
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content)-1; i += 2 {
			labels[node.Content[i].Value] = node.Content[i+1].Value
		}
	default:
		return nil, fmt.Errorf("expected list or map for labels")
	}
	return labels, nil
}

func parseServiceNetworks(node *yaml.Node) (ServiceNetworks, error) {
	nets := make(ServiceNetworks)

	switch node.Kind {
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("expected string in networks list")
			}
			nets[item.Value] = nil
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content)-1; i += 2 {
			name := node.Content[i].Value
			var cfg ServiceNetworkConfig
			if node.Content[i+1].Kind == yaml.MappingNode {
				if err := node.Content[i+1].Decode(&cfg); err != nil {
					return nil, fmt.Errorf("parsing network config for %q: %w", name, err)
				}
				nets[name] = &cfg
			} else {
				nets[name] = nil
			}
		}
	default:
		return nil, fmt.Errorf("expected list or map for networks")
	}
	return nets, nil
}

func parseSysctls(node *yaml.Node) (Sysctls, error) {
	sc := make(Sysctls)

	switch node.Kind {
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("expected string in sysctls list")
			}
			k, v := parseEnvEntry(item.Value)
			sc[k] = v
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content)-1; i += 2 {
			sc[node.Content[i].Value] = node.Content[i+1].Value
		}
	default:
		return nil, fmt.Errorf("expected list or map for sysctls")
	}
	return sc, nil
}

func parseStringList(node *yaml.Node) (StringList, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return StringList{node.Value}, nil
	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return nil, err
		}
		return StringList(list), nil
	default:
		return nil, fmt.Errorf("expected string or list")
	}
}

func parseBuild(node *yaml.Node) (*BuildConfig, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		return &BuildConfig{Context: node.Value}, nil
	case yaml.MappingNode:
		var b BuildConfig
		if err := node.Decode(&b); err != nil {
			return nil, err
		}
		return &b, nil
	default:
		return nil, fmt.Errorf("expected string or map for build")
	}
}
