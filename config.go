package libsquash

import (
	"strings"
	"time"
)

type containerConfig struct {
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

type config struct {
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

type layerConfig struct {
	ID                string           `json:"id"`
	Parent            string           `json:"parent,omitempty"`
	Comment           string           `json:"comment"`
	Created           time.Time        `json:"created"`
	V1ContainerConfig *containerConfig `json:"ContainerConfig,omitempty"`  // Docker 1.0.0, 1.0.1
	V2ContainerConfig *containerConfig `json:"container_config,omitempty"` // All other versions
	Container         string           `json:"container"`
	Config            *config          `json:"config,omitempty"`
	DockerVersion     string           `json:"docker_version"`
	Architecture      string           `json:"architecture"`
}

func (l *layerConfig) ContainerConfig() *containerConfig {
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

	l.V2ContainerConfig = &containerConfig{}

	return l.V2ContainerConfig
}

// Port is a type for representing docker port mappings
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
