package libsquash

import (
	"archive/tar"
	"strings"
)

/*
Layer represents a layer inside the docker image. A Layer consists of a struct
representation of the json config and placeholders for the tar headers for the
various required files
*/
type Layer struct {
	// LayerConfig is the layer config json
	LayerConfig *LayerConfig

	// DirHeader is the header for <uuid>/
	DirHeader *tar.Header

	// VersionHeader is the header for <uuid>/VERSION
	VersionHeader *tar.Header

	// JSONHeader is the header for <uuid>/json
	JSONHeader *tar.Header

	// LayerTarHeader is the header for <uuid>/layer.tar
	LayerTarHeader *tar.Header
}

// Cmd is a convenience function that prints out the command for layer "l". The
// command is shortened to be no more than 60 characters
func (l *Layer) Cmd() string {
	ret := strings.Join(l.LayerConfig.ContainerConfig().Cmd, " ")
	if len(ret) > 60 {
		ret = ret[:60]
	}
	return ret
}

// Clone returns a copy of layer l
func (l *Layer) Clone() *Layer {
	return &Layer{
		LayerConfig:    l.LayerConfig,
		DirHeader:      l.DirHeader,
		VersionHeader:  l.VersionHeader,
		JSONHeader:     l.JSONHeader,
		LayerTarHeader: l.LayerTarHeader,
	}
}
