package libsquash

import (
	"strings"
	"time"
)

type ContainerConfig struct {
	AttachStderr    bool
	AttachStdin     bool
	AttachStdout    bool
	Cmd             []string
	CpuShares       int64
	Dns             []string
	Domainname      string
	Entrypoint      []string
	Env             []string
	Hostname        string
	Image           string
	Memory          int64
	MemorySwap      int64
	NetworkDisabled bool
	OnBuild         []string
	OpenStdin       bool
	PortSpecs       []string
	StdinOnce       bool
	Tty             bool
	User            string
	Volumes         map[string]struct{}
	VolumesFrom     string
}

type Config struct {
	AttachStderr    bool
	AttachStdin     bool
	AttachStdout    bool
	Cmd             []string
	CpuShares       int64
	Dns             []string // For Docker API v1.9 and below only
	Domainname      string
	Entrypoint      []string
	Env             []string
	ExposedPorts    map[Port]struct{}
	Hostname        string
	Image           string
	Memory          int64
	MemorySwap      int64
	NetworkDisabled bool
	OnBuild         []string
	OpenStdin       bool
	PortSpecs       []string
	StdinOnce       bool
	Tty             bool
	User            string
	Volumes         map[string]struct{}
	VolumesFrom     string
	WorkingDir      string
}

type LayerConfig struct {
	Id                string           `json:"id"`
	Parent            string           `json:"parent,omitempty"`
	Comment           string           `json:"comment"`
	Created           time.Time        `json:"created"`
	V1ContainerConfig *ContainerConfig `json:"ContainerConfig,omitempty"`  // Docker 1.0.0, 1.0.1
	V2ContainerConfig *ContainerConfig `json:"container_config,omitempty"` // All other versions
	Container         string           `json:"container"`
	Config            *Config          `json:"config,omitempty"`
	DockerVersion     string           `json:"docker_version"`
	Architecture      string           `json:"architecture"`
}

func (l *LayerConfig) ContainerConfig() *ContainerConfig {
	if l.V2ContainerConfig != nil {
		return l.V2ContainerConfig
	}

	// If the exports use the 1.0.x json field name, convert it to the newer field
	// name which appears to work in all versions.
	if l.V1ContainerConfig != nil {
		l.V2ContainerConfig = l.V1ContainerConfig
		l.V1ContainerConfig = nil
		return l.V2ContainerConfig
	}

	l.V2ContainerConfig = &ContainerConfig{}

	return l.V2ContainerConfig
}

type Port string

// Port returns the number of the port.
func (p Port) Port() string {
	return strings.Split(string(p), "/")[0]
}

// Proto returns the name of the protocol.
func (p Port) Proto() string {
	parts := strings.Split(string(p), "/")
	if len(parts) == 1 {
		return "tcp"
	}
	return parts[1]
}

func augmentSquashed(s, l *LayerConfig) *ContainerConfig {
	squashed := s.ContainerConfig()
	last := l.ContainerConfig()

	if len(squashed.Hostname) == 0 {
		squashed.Hostname = last.Hostname
	}
	if len(squashed.Domainname) == 0 {
		squashed.Domainname = last.Domainname
	}
	if len(squashed.Entrypoint) == 0 {
		squashed.Entrypoint = last.Entrypoint
	}
	if len(squashed.User) == 0 {
		squashed.User = last.User
	}
	if squashed.Memory == 0 {
		squashed.Memory = last.Memory
	}
	if squashed.MemorySwap == 0 {
		squashed.MemorySwap = last.MemorySwap
	}
	if squashed.CpuShares == 0 {
		squashed.CpuShares = last.CpuShares
	}

	squashed.AttachStdin = squashed.AttachStdin || last.AttachStdin
	squashed.AttachStdout = squashed.AttachStdout || last.AttachStdout
	squashed.AttachStderr = squashed.AttachStderr || last.AttachStderr

	if len(squashed.PortSpecs) == 0 {
		squashed.PortSpecs = last.PortSpecs
	}

	squashed.Tty = squashed.Tty || last.Tty
	squashed.OpenStdin = squashed.OpenStdin || last.OpenStdin
	squashed.StdinOnce = squashed.StdinOnce || last.StdinOnce
	squashed.NetworkDisabled = squashed.NetworkDisabled || last.NetworkDisabled

	if len(squashed.OnBuild) == 0 {
		squashed.OnBuild = last.OnBuild
	}
	if len(squashed.Env) == 0 {
		squashed.Env = last.Env
	}
	if len(squashed.Volumes) == 0 {
		squashed.Volumes = last.Volumes
	}
	if len(squashed.VolumesFrom) == 0 {
		squashed.VolumesFrom = last.VolumesFrom
	}

	return squashed
}

func augmentSquashedConfig(s, l *LayerConfig) *Config {
	squashed := s.Config
	if squashed == nil {
		squashed = &Config{}
	}
	last := l.Config

	if len(squashed.Hostname) == 0 {
		squashed.Hostname = last.Hostname
	}
	if len(squashed.Domainname) == 0 {
		squashed.Domainname = last.Domainname
	}
	if len(squashed.Entrypoint) == 0 {
		squashed.Entrypoint = last.Entrypoint
	}
	if len(squashed.User) == 0 {
		squashed.User = last.User
	}
	if squashed.Memory == 0 {
		squashed.Memory = last.Memory
	}
	if squashed.MemorySwap == 0 {
		squashed.MemorySwap = last.MemorySwap
	}
	if squashed.CpuShares == 0 {
		squashed.CpuShares = last.CpuShares
	}

	squashed.AttachStdin = squashed.AttachStdin || last.AttachStdin
	squashed.AttachStdout = squashed.AttachStdout || last.AttachStdout
	squashed.AttachStderr = squashed.AttachStderr || last.AttachStderr

	if len(squashed.PortSpecs) == 0 {
		squashed.PortSpecs = last.PortSpecs
	}

	squashed.Tty = squashed.Tty || last.Tty
	squashed.OpenStdin = squashed.OpenStdin || last.OpenStdin
	squashed.StdinOnce = squashed.StdinOnce || last.StdinOnce
	squashed.NetworkDisabled = squashed.NetworkDisabled || last.NetworkDisabled

	if len(squashed.OnBuild) == 0 {
		squashed.OnBuild = last.OnBuild
	}
	if len(squashed.Env) == 0 {
		squashed.Env = last.Env
	}
	if len(squashed.Volumes) == 0 {
		squashed.Volumes = last.Volumes
	}
	if len(squashed.VolumesFrom) == 0 {
		squashed.VolumesFrom = last.VolumesFrom
	}

	if len(squashed.WorkingDir) == 0 {
		squashed.WorkingDir = last.WorkingDir
	}

	if len(squashed.ExposedPorts) == 0 {
		squashed.ExposedPorts = last.ExposedPorts
	}

	return squashed
}
