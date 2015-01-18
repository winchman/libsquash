package libsquash

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/winchman/libsquash/tarball"
)

var uuidRegex = regexp.MustCompile("^[a-f0-9]{64}$")

func (e *export) IngestImageMetadata(tarstream io.Reader) error {
	if err := tarball.Walk(tarstream, func(t *tarball.TarFile) error {
		switch Type(t) {
		case Ignore:
			return nil
		case Repositories:
			if err := json.NewDecoder(t.Stream).Decode(&e.Repositories); err != nil {
				return err
			}
			// Export may have multiple branches with the same parent; if so, abort.
			for _, v := range e.Repositories {
				commits := map[string]string{}
				for tag, commit := range *v {
					commits[commit] = tag
				}
				if len(commits) > 1 {
					return errorMultipleBranchesSameParent
				}
			}
		case JSON:
			uuid := t.NameParts()[0]
			if e.Layers[uuid] == nil {
				e.Layers[uuid] = &layer{}
			}
			e.Layers[uuid].JSONHeader = t.Header
			if err := json.NewDecoder(t.Stream).Decode(&e.Layers[uuid].LayerConfig); err != nil {
				return err
			}
		case LayerTar:
			uuid := t.NameParts()[0]
			if e.Layers[uuid] == nil {
				e.Layers[uuid] = &layer{}
			}
			if err := tarball.Walk(t.Stream, func(tf *tarball.TarFile) error {
				filePath := nameWithoutWhiteoutPrefix(tf.Name())
				if e.fileToLayers[filePath] == nil {
					e.fileToLayers[filePath] = []fileLoc{}
				}
				foundWhiteout := isWhiteout(tf.Name())
				e.fileToLayers[filePath] = append(e.fileToLayers[filePath], fileLoc{
					uuid:     uuid,
					whiteout: foundWhiteout,
				})

				if foundWhiteout {
					e.whiteouts = append(e.whiteouts, whiteoutFile{
						prefix: filePath,
						uuid:   uuid,
					})
				}
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return e.populateFileData()
}

func (e *export) populateFileData() error {

	e.start = e.FirstSquash()

	// Can't find a previously squashed layer, default to root
	if e.start == nil {
		e.start = e.Root()
	}

	if e.start == nil {
		return errorNoFROM
	}

	index := 0
	current := e.start
	orderMap := map[string]int{}
	for {
		orderMap[current.LayerConfig.ID] = index
		index++
		current = e.ChildOf(current.LayerConfig.ID)
		if current == nil {
			break
		}
	}

	for path, fileLocs := range e.fileToLayers {
		if len(fileLocs) > 0 {
			greatest := fileLocs[0]
			for _, loc := range fileLocs {
				if orderMap[loc.uuid] > orderMap[greatest.uuid] {
					greatest = loc
				}
			}
			if e.layerToFiles[greatest.uuid] == nil {
				e.layerToFiles[greatest.uuid] = map[string]bool{}
			}

			// if name matches whiteout prefix and the whiteout file is found
			// in a layer that is >= greatest.uuid, skip
			uuidContainingWhiteout, matches := matchesWhiteout(path, e.whiteouts)
			if (matches && orderMap[uuidContainingWhiteout] >= orderMap[greatest.uuid]) || greatest.whiteout {
				delete(e.layerToFiles[greatest.uuid], path)
			} else {
				e.layerToFiles[greatest.uuid][path] = true
			}
		}
	}

	return nil
}

func (e *export) SquashLayers(into, from *layer, tarstream io.Reader, outstream io.Writer) (imageID string, err error) {
	tempfile, err := ioutil.TempFile("", "libsquash")
	if err != nil {
		return "", err
	}

	defer func() {
		_ = tempfile.Close()
		_ = os.RemoveAll(tempfile.Name())
	}()

	var squashLayerTarWriter = tar.NewWriter(tempfile)

	if err = tarball.Walk(tarstream, func(t *tarball.TarFile) error {
		nameParts := t.NameParts()
		switch Type(t) {
		case Directory:
			uuidPart := nameParts[0]
			e.Layers[uuidPart].DirHeader = t.Header
		case LayerTar:
			uuidPart := nameParts[0]
			e.Layers[uuidPart].LayerTarHeader = t.Header
			if err := tarball.Walk(t.Stream, func(tf *tarball.TarFile) error {
				filePath := nameWithoutWhiteoutPrefix(tf.Name())
				if e.layerToFiles[uuidPart][filePath] {
					squashLayerTarWriter.WriteHeader(tf.Header)
					if _, err := io.Copy(squashLayerTarWriter, tf.Stream); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				return err
			}
		case Version:
			uuidPart := nameParts[0]
			e.Layers[uuidPart].VersionHeader = t.Header
		}
		return nil
	}); err != nil {
		return "", err
	}

	debugf("Squashing from %s into %s\n", from.LayerConfig.ID[:12], into.LayerConfig.ID[:12])

	if err := squashLayerTarWriter.Close(); err != nil {
		return "", err
	}

	if _, err = tempfile.Seek(0, 0); err != nil {
		return "", err
	}

	debug("  -  Rewriting child history")
	if err := e.rewriteChildren(into); err != nil {
		return "", err
	}

	var tw = tar.NewWriter(outstream)
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
		if err := tw.WriteHeader(dir); err != nil {
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
		if err := tw.WriteHeader(version); err != nil {
			return "", err
		}
		if _, err := tw.Write([]byte("1.0")); err != nil {
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
		if err := tw.WriteHeader(jsonHdr); err != nil {
			return "", err
		}
		if _, err := tw.Write(jsonBytes); err != nil {
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
			fi, err := tempfile.Stat()
			if err != nil {
				return "", err
			}
			layerTar.Size = fi.Size()
			tw.WriteHeader(layerTar)
			if _, err := io.Copy(tw, tempfile); err != nil {
				return "", nil
			}
		} else {
			layerTar.Size = 1024
			tw.WriteHeader(layerTar)
			tw.Write(bytes.Repeat([]byte("\x00"), 1024))
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

func (e *export) rewriteChildren(from *layer) error {
	var entry = from

	squashID := entry.LayerConfig.ID
	for {
		if entry == nil {
			break
		}

		child := e.ChildOf(entry.LayerConfig.ID)

		/*
			if the layer is not the squash layer
				if: we have a #(nop) that is not an ADD, skip it
				else: remove the stuff in the layer.tar
		*/
		if entry.LayerConfig.ID != squashID && !strings.Contains(entry.Cmd(), "#(squash)") {
			if strings.Contains(entry.Cmd(), "#(nop)") && !strings.Contains(entry.Cmd(), "ADD") {
				if err := e.ReplaceLayer(entry); err != nil {
					return err
				}
			} else {
				e.RemoveLayer(entry)
			}
		}

		entry = child
	}
	return nil
}
