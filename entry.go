package libsquash

import (
	"archive/tar"
	"bytes"
	"time"
)

type exportedImage struct {
	LayerConfig    *LayerConfig
	LayerTarBuffer bytes.Buffer
	DirHeader      *tar.Header
	VersionHeader  *tar.Header
	JsonHeader     *tar.Header
	LayerTarHeader *tar.Header
}

func newLayerConfig(id, parent, comment string) *LayerConfig {
	return &LayerConfig{
		Id:            id,
		Parent:        parent,
		Comment:       comment,
		Created:       time.Now().UTC(),
		DockerVersion: "0.1.2",
		Architecture:  "x86_64",
	}
}
