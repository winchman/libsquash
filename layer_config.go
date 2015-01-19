package libsquash

import (
	"time"
)

// LayerConfig is the top-level json config for ar docker layer
type LayerConfig struct {
	ID                string           `json:"id"`
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

// NewLayerConfig produces an empty LayerConfig, initialized with a few
// sensible defaults
func NewLayerConfig(id, parent, comment string) *LayerConfig {
	return &LayerConfig{
		ID:            id,
		Parent:        parent,
		Comment:       comment,
		Created:       time.Now().UTC(),
		DockerVersion: "0.1.2",
		Architecture:  "x86_64",
	}
}

// ContainerConfig is a sub config of LayerConfig
type ContainerConfig struct {
	AttachStderr    bool
	AttachStdin     bool
	AttachStdout    bool
	Cmd             []string
	CPUShares       int64    `json:"CpuShares"`
	DNS             []string `json:"Dns"`
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

// Config is a sub config of LayerConfig
type Config struct {
	AttachStderr    bool
	AttachStdin     bool
	AttachStdout    bool
	Cmd             []string
	CPUShares       int64    `json:"CpuShares"`
	DNS             []string `json:"Dns"` // For Docker API v1.9 and below only
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

// ContainerConfig normalizes the V1 and V2ContainerConfig and returns the
// correct version
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
