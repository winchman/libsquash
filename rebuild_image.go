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

	tw := tarball.NewTarstream(outstream)

	var latestDirHeader, latestVersionHeader, latestJSONHeader, latestTarHeader *tar.Header

	squashedLayerConfig := into.LayerConfig

	current := e.Root()
	order := []*layer{} // TODO: optimize, remove this
	for {
		order = append(order, current)
		current = e.ChildOf(current.LayerConfig.ID)
		if current == nil {
			break
		}
	}

	var retID string

	for index, current := range order {
		var dir = current.DirHeader
		if latestDirHeader == nil {
			latestDirHeader = dir
		}
		if dir == nil {
			dir = latestDirHeader
		}
		dir.Name = current.LayerConfig.ID + "/"
		if err := tw.Add(&tarball.TarFile{Header: dir}); err != nil {
			return "", err
		}

		var version = current.VersionHeader
		if latestVersionHeader == nil {
			latestVersionHeader = version
		}
		if version == nil {
			version = latestVersionHeader
		}
		version.Name = current.LayerConfig.ID + "/VERSION"

		if err := tw.Add(&tarball.TarFile{
			Header: version,
			Stream: bytes.NewBuffer([]byte("1.0")),
		}); err != nil {
			return "", err
		}

		var jsonHdr = current.JSONHeader
		if latestJSONHeader == nil {
			latestJSONHeader = jsonHdr
		}
		if jsonHdr == nil {
			jsonHdr = latestJSONHeader
		}
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

		if err := tw.Add(&tarball.TarFile{
			Header: jsonHdr,
			Stream: bytes.NewBuffer(jsonBytes),
		}); err != nil {
			return "", err
		}

		var layerTar = current.LayerTarHeader
		if latestTarHeader == nil {
			latestTarHeader = layerTar
		}
		if layerTar == nil {
			layerTar = latestTarHeader
		}

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

		if index == len(order)-1 {
			retID = current.LayerConfig.ID
		}
	}

	if err := tw.Close(); err != nil {
		return "", err
	}

	return retID, nil
}
