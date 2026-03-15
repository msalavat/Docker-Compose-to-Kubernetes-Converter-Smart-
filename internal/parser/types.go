package parser

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
	Image         string            `yaml:"image,omitempty"`
	ContainerName string            `yaml:"container_name,omitempty"`
	Ports         []PortConfig      `yaml:"-"`
	Volumes       []VolumeMount     `yaml:"-"`
	Environment   map[string]string `yaml:"-"`
	EnvFile       StringList        `yaml:"env_file,omitempty"`
	DependsOn     DependsOn         `yaml:"-"`
	Healthcheck   *HealthcheckConfig `yaml:"healthcheck,omitempty"`
	Deploy        *DeployConfig     `yaml:"deploy,omitempty"`
	Command       StringList        `yaml:"-"`
	Entrypoint    StringList        `yaml:"-"`
	Labels        map[string]string `yaml:"-"`
	Networks      ServiceNetworks   `yaml:"-"`
	Expose        []string          `yaml:"expose,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
	User          string            `yaml:"user,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty"`
	StdinOpen     bool              `yaml:"stdin_open,omitempty"`
	Tty           bool              `yaml:"tty,omitempty"`
	Privileged    bool              `yaml:"privileged,omitempty"`
	ReadOnly      bool              `yaml:"read_only,omitempty"`
	CapAdd        []string          `yaml:"cap_add,omitempty"`
	CapDrop       []string          `yaml:"cap_drop,omitempty"`
	SecurityOpt   []string          `yaml:"security_opt,omitempty"`
	Sysctls       map[string]string `yaml:"-"`
	ExtraHosts    []string          `yaml:"extra_hosts,omitempty"`
	DNS           StringList        `yaml:"-"`
	DNSSearch     StringList        `yaml:"-"`
	Logging       *LoggingConfig    `yaml:"logging,omitempty"`
	Build         *BuildConfig      `yaml:"-"`
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
	Condition string // "service_started", "service_healthy", "service_completed_successfully"
}

// HealthcheckConfig represents a service healthcheck.
type HealthcheckConfig struct {
	Test        StringList `yaml:"test,omitempty"`
	Interval    string     `yaml:"interval,omitempty"`
	Timeout     string     `yaml:"timeout,omitempty"`
	Retries     int        `yaml:"retries,omitempty"`
	StartPeriod string     `yaml:"start_period,omitempty"`
	Disable     bool       `yaml:"disable,omitempty"`
}

// DeployConfig represents deploy configuration.
type DeployConfig struct {
	Replicas      *int              `yaml:"replicas,omitempty"`
	Resources     *Resources        `yaml:"resources,omitempty"`
	RestartPolicy *RestartPolicy    `yaml:"restart_policy,omitempty"`
	Placement     *Placement        `yaml:"placement,omitempty"`
	Labels        map[string]string `yaml:"-"`
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

// StringList is a type that can unmarshal from either a string or a list of strings.
type StringList []string

// ServiceNetworks can unmarshal from a list of network names or a map of network configs.
type ServiceNetworks map[string]*ServiceNetworkConfig

// ServiceNetworkConfig represents per-service network configuration.
type ServiceNetworkConfig struct {
	Aliases []string `yaml:"aliases,omitempty"`
}
