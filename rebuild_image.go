package libsquash

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"

	"github.com/winchman/libsquash/tarball"
)

var uuidRegex = regexp.MustCompile("^[a-f0-9]{64}$")

func (e *export) RebuildImage(into *layer, outstream io.Writer, squashLayerFile *os.File) (imageID string, err error) {
	var (
		latestDirHeader, latestVersionHeader *tar.Header
		latestJSONHeader, latestTarHeader    *tar.Header
		retID                                string
	)

	tw := tarball.NewTarstream(outstream)
	squashedLayerConfig := into.LayerConfig
	current := e.Root()

	for {
		var dir *tar.Header
		dir, latestDirHeader = chooseDefault(current.DirHeader, latestDirHeader)
		dir.Name = current.LayerConfig.ID + "/"
		if err := tw.Add(&tarball.TarFile{Header: dir}); err != nil {
			return "", err
		}

		var version *tar.Header
		version, latestVersionHeader = chooseDefault(current.VersionHeader, latestVersionHeader)
		version.Name = current.LayerConfig.ID + "/VERSION"
		if err := tw.Add(&tarball.TarFile{Header: version, Stream: bytes.NewBuffer([]byte("1.0"))}); err != nil {
			return "", err
		}

		var jsonHdr *tar.Header
		jsonHdr, latestJSONHeader = chooseDefault(current.JSONHeader, latestJSONHeader)
		jsonHdr.Name = current.LayerConfig.ID + "/json"

		var jsonBytes []byte
		var err error
		if current.LayerConfig.ID == into.LayerConfig.ID {
			jsonBytes, err = json.Marshal(squashedLayerConfig)
		} else {
			jsonBytes, err = json.Marshal(current.LayerConfig)
		}
		if err != nil {
			return "", err
		}
		jsonHdr.Size = int64(len(jsonBytes))
		if err := tw.Add(&tarball.TarFile{Header: jsonHdr, Stream: bytes.NewBuffer(jsonBytes)}); err != nil {
			return "", err
		}

		var layerTar *tar.Header
		layerTar, latestTarHeader = chooseDefault(current.LayerTarHeader, latestTarHeader)
		layerTar.Name = current.LayerConfig.ID + "/layer.tar"

		if current.LayerConfig.ID == into.LayerConfig.ID {
			fi, err := squashLayerFile.Stat()
			if err != nil {
				return "", err
			}
			layerTar.Size = fi.Size()
			if err := tw.Add(&tarball.TarFile{Header: layerTar, Stream: squashLayerFile}); err != nil {
				return "", err
			}
		} else {
			layerTar.Size = 1024
			if err := tw.Add(
				&tarball.TarFile{Header: layerTar, Stream: bytes.NewBuffer(bytes.Repeat([]byte("\x00"), 1024))},
			); err != nil {
				return "", err
			}
		}

		child := e.ChildOf(current.LayerConfig.ID)
		if child == nil {
			retID = current.LayerConfig.ID
			break
		}
		current = child
	}
	if err := tw.Close(); err != nil {
		return "", err
	}

	return retID, nil
}

func chooseDefault(alpha, beta *tar.Header) (*tar.Header, *tar.Header) {
	if beta == nil {
		beta = alpha
	}
	if alpha == nil {
		alpha = beta
	}
	return alpha, beta
}
