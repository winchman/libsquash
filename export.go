package libsquash

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
)

var uuidRegex = regexp.MustCompile("^[a-f0-9]{64}$")

type TagInfo map[string]string

type export struct {
	Entries      map[string]*exportedImage
	Repositories map[string]*TagInfo
	fileToLayers map[string][]fileLoc
	layerToFiles map[string]map[string]bool
	start        *exportedImage
	whiteouts    []whiteoutFile
}

type fileLoc struct {
	uuid     string
	whiteout bool // to indicate that the file as presented in this layer is a whiteout instead of a regular file
}

type whiteoutFile struct {
	uuid   string // the layer in which the whiteout was found
	prefix string // the file name prefix (without the ".wh." part)
}

func newExport() *export {
	return &export{
		Entries:      map[string]*exportedImage{},
		Repositories: map[string]*TagInfo{},
		fileToLayers: map[string][]fileLoc{},
		layerToFiles: map[string]map[string]bool{},
		whiteouts:    []whiteoutFile{},
	}
}

func (e *export) matchesWhiteout(filename string) (uuidContainingWhiteout string, matches bool) {
	for _, whiteout := range e.whiteouts {
		if strings.HasPrefix(filename, whiteout.prefix) {
			return whiteout.uuid, true
		}
	}
	return "", false
}

func (e *export) firstLayer(pattern string) *exportedImage {
	root := e.Root()
	for {
		if root == nil {
			break
		}

		cmd := strings.Join(root.LayerConfig.ContainerConfig().Cmd, " ")
		if strings.Contains(cmd, pattern) {
			break
		}
		root = e.ChildOf(root.LayerConfig.Id)
	}
	return root
}

func (e *export) FirstFrom() *exportedImage {
	return e.firstLayer("#(nop) ADD file")
}

func (e *export) FirstSquash() *exportedImage {
	return e.firstLayer("#(squash)")
}

// Root returns the top layer in the export
func (e *export) Root() *exportedImage {
	return e.ChildOf("")
}

func (e *export) LastChild() *exportedImage {
	c := e.Root()
	for {
		if e.ChildOf(c.LayerConfig.Id) == nil {
			break
		}
		c = e.ChildOf(c.LayerConfig.Id)
	}
	return c
}

// ChildOf returns the child layer or nil of the parent
func (e *export) ChildOf(parent string) *exportedImage {
	for _, entry := range e.Entries {
		if entry.LayerConfig.Parent == parent {
			return entry
		}
	}
	return nil
}

// GetById returns an exportedImaged with a prefix matching ID.  An error
// is returned multiple exportedImages matched.
func (e *export) GetById(idPrefix string) (*exportedImage, error) {
	matches := []*exportedImage{}
	for id, entry := range e.Entries {
		if strings.HasPrefix(id, idPrefix) {
			matches = append(matches, entry)
		}
	}

	if len(matches) > 1 {
		return nil, errors.New(fmt.Sprintf("%s is ambiguous. %d matched.", idPrefix, len(matches)))
	}

	if len(matches) == 0 {
		return nil, nil
	}

	return matches[0], nil
}

func (e *export) InsertLayer(parent string) (*exportedImage, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	layerConfig := newLayerConfig(id, parent, "squashed w/ docker-squash")
	layerConfig.ContainerConfig().Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(squash) from %s", parent[:12])}

	entry := &exportedImage{
		LayerConfig: layerConfig,
	}

	entry.LayerConfig.Created = time.Now().UTC()

	// rewrite child json
	child := e.ChildOf(parent)
	child.LayerConfig.Parent = id

	e.Entries[id] = entry

	return entry, err
}

func (e *export) ReplaceLayer(oldId string) (*exportedImage, error) {

	id, err := newID()
	if err != nil {
		return nil, err
	}

	orig := e.Entries[oldId]
	child := e.ChildOf(oldId)

	cmd := strings.Join(orig.LayerConfig.ContainerConfig().Cmd, " ")

	debugf("  -  Replacing %s w/ new layer %s (%s)\n", oldId[:12], id[:12], cmd)
	if child != nil {
		child.LayerConfig.Parent = id
	}

	layerConfig := orig.LayerConfig
	layerConfig.Id = id

	entry := &exportedImage{
		LayerConfig: layerConfig,
	}
	entry.LayerConfig.Created = time.Now().UTC()

	e.Entries[id] = entry

	orig.LayerTarBuffer.Reset()
	delete(e.Entries, oldId)

	return entry, nil
}

func (e *export) parseLayerMetadata(instream io.Reader) error {
	var tarReader = tar.NewReader(instream)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if header.Name == "." || header.Name == ".." || header.Name == "./" {
			continue
		}

		nameParts := strings.Split(header.Name, string(os.PathSeparator))
		if len(nameParts) == 0 {
			continue
		}

		if len(nameParts) == 1 {
			if nameParts[0] == "repositories" {
				if err = json.NewDecoder(tarReader).Decode(&e.Repositories); err != nil {
					return err
				}
			}

			// Export may have multiple branches with the same parent.
			// We can't handle that currently so abort.
			for _, v := range e.Repositories {
				commits := map[string]string{}
				for tag, commit := range *v {
					commits[commit] = tag
				}
				if len(commits) > 1 {
					return errorMultipleBranchesSameParent
				}
			}

			continue
		}

		uuidPart := nameParts[0]
		fileName := nameParts[1]
		if e.Entries[uuidPart] == nil {
			e.Entries[uuidPart] = &exportedImage{}
		}

		switch fileName {
		case "json":
			e.Entries[uuidPart].JsonHeader = header
			if err = json.NewDecoder(tarReader).Decode(&e.Entries[uuidPart].LayerConfig); err != nil {
				return err
			}
		case "layer.tar":
			layerReader := tar.NewReader(tarReader)
			for {
				fileHeader, err := layerReader.Next()
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
				filePath := nameWithoutWhiteoutPrefix(fileHeader.Name)
				if e.fileToLayers[filePath] == nil {
					e.fileToLayers[filePath] = []fileLoc{}
				}
				foundWhiteout := isWhiteout(fileHeader.Name)
				e.fileToLayers[filePath] = append(e.fileToLayers[filePath], fileLoc{
					uuid:     uuidPart,
					whiteout: foundWhiteout,
				})

				if foundWhiteout {
					e.whiteouts = append(e.whiteouts, whiteoutFile{
						prefix: filePath,
						uuid:   uuidPart,
					})
				}
			}
		}
	}

	e.start = e.FirstSquash()
	// Can't find a previously squashed layer, use first ADD
	if e.start == nil {
		e.start = e.FirstFrom()
	}
	// Can't find a FROM, default to root
	if e.start == nil {
		e.start = e.Root()
	}

	if e.start == nil {
		return errorNoFROM
	}

	// TODO: optimize creation of ordered list - currently n^2, can be n
	index := 0
	current := e.start
	orderMap := map[string]int{}
	for {
		orderMap[current.LayerConfig.Id] = index
		index++
		current = e.ChildOf(current.LayerConfig.Id)
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
			/*
				if name matches whiteout prefix and the whiteout file is found in a layer that is >= greatest.uuid, skip
			*/
			uuidContainingWhiteout, matches := e.matchesWhiteout(path)
			if (matches && orderMap[uuidContainingWhiteout] >= orderMap[greatest.uuid]) || greatest.whiteout {
				delete(e.layerToFiles[greatest.uuid], path)
			} else {
				e.layerToFiles[greatest.uuid][path] = true
			}
		}
	}

	return nil
}

func (e *export) SquashLayers(into, from *exportedImage, instream io.Reader, outstream io.Writer) error {
	var squashLayerTarWriter = tar.NewWriter(&into.LayerTarBuffer)

	var tarReader = tar.NewReader(instream)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if header.Name == "." || header.Name == ".." || header.Name == "./" {
			continue
		}

		nameParts := strings.Split(header.Name, string(os.PathSeparator))
		if len(nameParts) == 0 {
			continue
		}

		uuidPart := nameParts[0]
		fileName := nameParts[1]

		switch fileName {
		case "":
			e.Entries[uuidPart].DirHeader = header
		case "layer.tar":
			e.Entries[uuidPart].LayerTarHeader = header
			layerReader := tar.NewReader(tarReader)
			for {
				fileHeader, err := layerReader.Next()
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
				filePath := nameWithoutWhiteoutPrefix(fileHeader.Name)
				if e.layerToFiles[uuidPart][filePath] {
					fileBytes, err := ioutil.ReadAll(layerReader) //TODO: get rid of ReadAll if possible
					if err != nil {
						return err
					}
					squashLayerTarWriter.WriteHeader(fileHeader)
					squashLayerTarWriter.Write(fileBytes)
				}
			}
		case "VERSION":
			e.Entries[uuidPart].VersionHeader = header
		}
	}

	debugf("Squashing from %s into %s\n", from.LayerConfig.Id[:12], into.LayerConfig.Id[:12])

	if err := squashLayerTarWriter.Close(); err != nil {
		return err
	}

	var tw = tar.NewWriter(outstream)
	var latestDirHeader, latestVersionHeader, latestJsonHeader, latestTarHeader *tar.Header

	debug("  -  Rewriting child history")
	if err := e.rewriteChildren(into); err != nil {
		return err
	}

	squashedLayerConfig := into.LayerConfig
	last := e.LastChild().LayerConfig

	squashedLayerConfig.V2ContainerConfig = augmentSquashed(squashedLayerConfig, last)
	squashedLayerConfig.Config = augmentSquashedConfig(squashedLayerConfig, last)

	current := from
	order := []*exportedImage{} // TODO: optimize, remove this
	for {
		order = append(order, current)
		current = e.ChildOf(current.LayerConfig.Id)
		if current == nil {
			break
		}
	}

	for _, current := range order {
		var dir = current.DirHeader
		if latestDirHeader == nil {
			latestDirHeader = dir
		}
		if dir == nil {
			dir = latestDirHeader
		}
		dir.Name = current.LayerConfig.Id + "/"
		if err := tw.WriteHeader(dir); err != nil {
			return err
		}

		var version = current.VersionHeader
		if latestVersionHeader == nil {
			latestVersionHeader = version
		}
		if version == nil {
			version = latestVersionHeader
		}
		version.Name = current.LayerConfig.Id + "/VERSION"
		if err := tw.WriteHeader(version); err != nil {
			return err
		}
		if _, err := tw.Write([]byte("1.0")); err != nil {
			return err
		}

		var jsonHdr = current.JsonHeader
		if latestJsonHeader == nil {
			latestJsonHeader = jsonHdr
		}
		if jsonHdr == nil {
			jsonHdr = latestJsonHeader
		}
		jsonHdr.Name = current.LayerConfig.Id + "/json"

		var jsonBytes []byte
		var err error
		if current.LayerConfig.Id == into.LayerConfig.Id {
			jsonBytes, err = json.Marshal(squashedLayerConfig)
		} else {
			jsonBytes, err = json.Marshal(current.LayerConfig)
		}
		if err != nil {
			return err
		}
		jsonHdr.Size = int64(len(jsonBytes))
		if err := tw.WriteHeader(jsonHdr); err != nil {
			return err
		}
		if _, err := tw.Write(jsonBytes); err != nil {
			return err
		}

		var layerTar = current.LayerTarHeader
		if latestTarHeader == nil {
			latestTarHeader = layerTar
		}
		if layerTar == nil {
			layerTar = latestTarHeader
		}
		layerTar.Name = current.LayerConfig.Id + "/layer.tar"
		layerTar.Size = int64(current.LayerTarBuffer.Len())
		tw.WriteHeader(layerTar)
		tw.Write(current.LayerTarBuffer.Bytes())
	}

	return tw.Close()
}

func (e *export) rewriteChildren(from *exportedImage) error {
	var entry = from

	squashId := entry.LayerConfig.Id
	for {
		if entry == nil {
			break
		}

		cmd := strings.Join(entry.LayerConfig.ContainerConfig().Cmd, " ")

		if entry.LayerConfig.Id == squashId || strings.Contains(cmd, "#(squash)") {
			entry = e.ChildOf(entry.LayerConfig.Id)
			continue
		}

		// if: we have a #(nop) that is not an ADD, skip it
		// else: remove the stuff in the layer.tar
		if strings.Contains(cmd, "#(nop)") && !strings.Contains(cmd, "ADD") {
			entry.LayerConfig.Created = time.Now().UTC()
			entry = e.ChildOf(entry.LayerConfig.Id)
		} else {
			debugf("  -  Removing %s. Squashed. (%s)\n", entry.LayerConfig.Id[:12], cmd)

			child := e.ChildOf(entry.LayerConfig.Id)
			if child != nil {
				child.LayerConfig.Parent = entry.LayerConfig.Parent
			}
			entry.LayerTarBuffer.Reset()
			delete(e.Entries, entry.LayerConfig.Id)
			entry = child
		}
	}
	return nil
}
