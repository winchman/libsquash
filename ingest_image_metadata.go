package libsquash

import (
	"encoding/json"
	"io"

	"github.com/winchman/libsquash/tarball"
)

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
