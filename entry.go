package libsquash

import (
	"archive/tar"
	"bytes"
	"time"
)

type exportedImage struct {
	LayerConfig    *layerConfig
	LayerTarBuffer bytes.Buffer
	DirHeader      *tar.Header
	VersionHeader  *tar.Header
	JSONHeader     *tar.Header
	LayerTarHeader *tar.Header
}

func newLayerConfig(id, parent, comment string) *layerConfig {
	return &layerConfig{
		ID:            id,
		Parent:        parent,
		Comment:       comment,
		Created:       time.Now().UTC(),
		DockerVersion: "0.1.2",
		Architecture:  "x86_64",
	}
}
