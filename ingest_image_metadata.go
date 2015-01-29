package libsquash

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/winchman/libsquash/tarball"
)

var (
	// ErrorMultipleBranchesSameParent is an edge case that libsquash does not currently handle
	ErrorMultipleBranchesSameParent = errors.New("this image is a full " +
		"repository export w/ multiple images in it. Please generate the export " +
		"from a specific image ID or tag.",
	)

	// ErrorNoFROM is returned when o root layer can be found
	ErrorNoFROM = errors.New("no root layer found")

	// ErrorInvalidFROM is returned when the From layer provided in
	// SquashOptions cannot be found in the image tarball
	ErrorInvalidFROM = errors.New("invalid/nonexistent From layer provided in SquashOptions")
)

/*
IngestImageMetadata walks the files in the "tarstream" tarball and checks for
several things:

1. check for the "repositories" file - if present, determine if it cancels the squash

2. check for each "json" file, read it into a data structure

3. check for each "layer.tar" file, noting which filesystem files are present
or deleted via aufs-style whiteout files (.wh..wh.<file>)

To determine what files should come from each layer.tar (which was the last to
modify that file), we build a list like so:

	fileToLayers:
	------------
	file1: []layer
	file2: []layer

After completing the processing of the tarball, this function calls
another that translates the list as follows (note: a uuid is the unique
identifier for a layer):

	layerToFiles:
	------------
	uuid1: file1 -> true, file3 -> true
	uuid2: file2 -> true

The next time we we iterate over the image tarball, we only have one layer.tar
at a time.  We need to know, based on the uuid of that layer.tar, which files
to pull from it. That requires the layerToFiles structure.
*/
func (e *Export) IngestImageMetadata(tarstream io.Reader) error {
	if err := tarball.Walk(tarstream, func(t *tarball.TarFile) error {
		switch ParseType(t) {
		case IgnoreType:
			// ignore
		case RepositoriesType:
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
					return ErrorMultipleBranchesSameParent
				}
			}
		case JSONType:
			uuid := t.NameParts()[0]
			if e.Layers[uuid] == nil {
				e.Layers[uuid] = &Layer{}
			}
			e.Layers[uuid].JSONHeader = t.Header
			if err := json.NewDecoder(t.Stream).Decode(&e.Layers[uuid].LayerConfig); err != nil {
				return err
			}
		case LayerTarType:
			uuid := t.NameParts()[0]
			if e.Layers[uuid] == nil {
				e.Layers[uuid] = &Layer{}
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
	return nil
}

// AssignStartLayer determines the layer at which to start the squashing
// process. If a From layer is provided in SquashOptions, that layer is
// preferred.  Next in line is the last squash layer, and if none exists, the
// root is used.  If the From layer in SquashOptions is invalid or the root
// layer cannot be found, AssignStartLayer will return an error
func (e *Export) AssignStartLayer(from string) error {
	// if a custom From layer is provided
	if from != "" {
		layer, err := e.GetByID(from)
		if err != nil {
			return err
		} else if layer == nil {
			return ErrorInvalidFROM
		} else {
			e.start = layer
			return nil
		}
	}

	e.start = e.FirstSquash()

	// Can't find a previously squashed layer, default to root
	if e.start == nil {
		e.start = e.Root()
	}

	if e.start == nil {
		return ErrorNoFROM
	}
	return nil
}

// PopulateFileData populates the layerToFiles as described in the comments for
// IngestImageMetadata
func (e *Export) PopulateFileData() error {
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
