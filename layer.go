package libsquash

import (
	"archive/tar"
	"strings"
)

type layer struct {
	LayerConfig    *layerConfig
	DirHeader      *tar.Header
	VersionHeader  *tar.Header
	JSONHeader     *tar.Header
	LayerTarHeader *tar.Header
}

func (l *layer) Cmd() string {
	ret := strings.Join(l.LayerConfig.ContainerConfig().Cmd, " ")
	if len(ret) > 60 {
		ret = ret[:60]
	}
	return ret
}
