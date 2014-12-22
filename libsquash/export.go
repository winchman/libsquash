package libsquash

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

var uuidRegex = regexp.MustCompile("^[a-f0-9]{64}$")

type TagInfo map[string]string

type Export struct {
	Entries      map[string]*ExportedImage
	Repositories map[string]*TagInfo
}

type Port string

// Port returns the number of the port.
func (p Port) Port() string {
	return strings.Split(string(p), "/")[0]
}

// Proto returns the name of the protocol.
func (p Port) Proto() string {
	parts := strings.Split(string(p), "/")
	if len(parts) == 1 {
		return "tcp"
	}
	return parts[1]
}

type ContainerConfig struct {
	Hostname        string
	Domainname      string
	Entrypoint      []string
	User            string
	Memory          int64
	MemorySwap      int64
	CpuShares       int64
	AttachStdin     bool
	AttachStdout    bool
	AttachStderr    bool
	PortSpecs       []string
	Tty             bool
	OpenStdin       bool
	StdinOnce       bool
	NetworkDisabled bool
	OnBuild         []string
	Env             []string
	Cmd             []string
	Dns             []string
	Image           string
	Volumes         map[string]struct{}
	VolumesFrom     string
}

type Config struct {
	Hostname        string
	Domainname      string
	User            string
	Memory          int64
	MemorySwap      int64
	CpuShares       int64
	AttachStdin     bool
	AttachStdout    bool
	AttachStderr    bool
	PortSpecs       []string
	ExposedPorts    map[Port]struct{}
	OnBuild         []string
	Tty             bool
	OpenStdin       bool
	StdinOnce       bool
	Env             []string
	Cmd             []string
	Dns             []string // For Docker API v1.9 and below only
	Image           string
	Volumes         map[string]struct{}
	VolumesFrom     string
	WorkingDir      string
	Entrypoint      []string
	NetworkDisabled bool
}

type LayerConfig struct {
	Id                string           `json:"id"`
	Parent            string           `json:"parent,omitempty"`
	Comment           string           `json:"comment"`
	Created           time.Time        `json:"created"`
	V1ContainerConfig *ContainerConfig `json:"ContainerConfig,omitempty"`  // Docker 1.0.0, 1.0.1
	V2ContainerConfig *ContainerConfig `json:"container_config,omitempty"` // All other versions
	Container         string           `json:"container"`
	Config            *Config          `json:"config,omitempty"`
	DockerVersion     string           `json:"docker_version"`
	Architecture      string           `json:"architecture"`
}

type tarFile struct {
	Contents *bytes.Buffer
	Header   *tar.Header
}

func (l *LayerConfig) ContainerConfig() *ContainerConfig {
	if l.V2ContainerConfig != nil {
		return l.V2ContainerConfig
	}

	// If the exports use the 1.0.x json field name, convert it to the newer field
	// name which appears to work in all versions.
	if l.V1ContainerConfig != nil {
		l.V2ContainerConfig = l.V1ContainerConfig
		l.V1ContainerConfig = nil
		return l.V2ContainerConfig
	}

	l.V2ContainerConfig = &ContainerConfig{}

	return l.V2ContainerConfig
}

func NewExport() *Export {
	return &Export{
		Entries:      map[string]*ExportedImage{},
		Repositories: map[string]*TagInfo{},
	}
}

func (e *Export) firstLayer(pattern string) *ExportedImage {
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

func (e *Export) FirstFrom() *ExportedImage {
	return e.firstLayer("#(nop) ADD file")
}

func (e *Export) FirstSquash() *ExportedImage {
	return e.firstLayer("#(squash)")
}

// Root returns the top layer in the export
func (e *Export) Root() *ExportedImage {
	return e.ChildOf("")
}

func (e *Export) LastChild() *ExportedImage {
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
func (e *Export) ChildOf(parent string) *ExportedImage {
	for _, entry := range e.Entries {
		if entry.LayerConfig.Parent == parent {
			return entry
		}
	}
	return nil
}

// GetById returns an ExportedImaged with a prefix matching ID.  An error
// is returned multiple ExportedImages matched.
func (e *Export) GetById(idPrefix string) (*ExportedImage, error) {
	matches := []*ExportedImage{}
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

func (e *Export) InsertLayer(parent string) (*ExportedImage, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	layerConfig := newLayerConfig(id, parent, "squashed w/ docker-squash")
	layerConfig.ContainerConfig().Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(squash) from %s", parent[:12])}

	entry := &ExportedImage{
		LayerConfig: layerConfig,
	}

	entry.LayerConfig.Created = time.Now().UTC()

	// rewrite child json
	child := e.ChildOf(parent)
	child.LayerConfig.Parent = id

	e.Entries[id] = entry

	return entry, err
}

func (e *Export) ReplaceLayer(oldId string) (*ExportedImage, error) {

	id, err := newID()
	if err != nil {
		return nil, err
	}

	orig := e.Entries[oldId]
	child := e.ChildOf(oldId)

	cmd := strings.Join(orig.LayerConfig.ContainerConfig().Cmd, " ")

	Debugf("  -  Replacing %s w/ new layer %s (%s)\n", oldId[:12], id[:12], cmd)
	if child != nil {
		child.LayerConfig.Parent = id
	}

	layerConfig := orig.LayerConfig
	layerConfig.Id = id

	entry := &ExportedImage{
		LayerConfig: layerConfig,
	}
	entry.LayerConfig.Created = time.Now().UTC()

	e.Entries[id] = entry

	delete(e.Entries, oldId)

	return entry, nil
}

// to => newEntry (squash layer), from => "start"
func (e *Export) SquashLayers(into, from *ExportedImage) (io.Reader, error) {
	Debugf("Squashing from %s into %s\n", from.LayerConfig.Id[:12], into.LayerConfig.Id[:12])

	var files = map[string]*tarFile{}
	var whiteouts = []string{}

	var current = from
	var order = []*ExportedImage{}
	for {
		order = append(order, current)
		current = e.ChildOf(current.LayerConfig.Id)
		if current == nil {
			break
		}
	}

	for _, current := range order {
		subtar := tar.NewReader(&current.LayerTarBuffer)

		for {
			header, err := subtar.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			nameParts := strings.Split(header.Name, string(os.PathSeparator))
			fileName := nameParts[len(nameParts)-1]
			// skip whiteout files
			if strings.HasPrefix(fileName, ".wh.") {
				whiteouts = append(whiteouts, header.Name)
				continue
			}

			if files[header.Name] == nil {
				files[header.Name] = &tarFile{
					Contents: new(bytes.Buffer),
					Header:   header,
				}
			}

			files[header.Name].Contents.Reset()
			if _, err = files[header.Name].Contents.ReadFrom(subtar); err != nil {
				return nil, err
			}
		}
	}

	files = deleteWhiteouts(files, whiteouts)

	// tar writer for layer.tar for new squash layer
	var squashLayerTarWriter = tar.NewWriter(&into.LayerTarBuffer)

	// create layer.tar for squash layer
	for _, file := range files {
		squashLayerTarWriter.WriteHeader(file.Header)
		squashLayerTarWriter.Write(file.Contents.Bytes())
	}
	if err := squashLayerTarWriter.Close(); err != nil {
		return nil, err
	}

	var out = new(bytes.Buffer)
	var tw = tar.NewWriter(out)
	var latestDirHeader, latestVersionHeader, latestJsonHeader, latestTarHeader *tar.Header

	Debug("  -  Rewriting child history")
	if err := e.rewriteChildren(into); err != nil {
		return nil, err
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
			return nil, err
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
			return nil, err
		}
		if _, err := tw.Write([]byte("1.0")); err != nil {
			return nil, err
		}

		var jsonHdr = current.JsonHeader
		if latestJsonHeader == nil {
			latestJsonHeader = jsonHdr
		}
		if jsonHdr == nil {
			jsonHdr = latestJsonHeader
		}
		jsonHdr.Name = current.LayerConfig.Id + "/json"
		jsonBytes, err := json.Marshal(current.LayerConfig)
		if err != nil {
			return nil, err
		}
		jsonHdr.Size = int64(len(jsonBytes))
		if err := tw.WriteHeader(jsonHdr); err != nil {
			return nil, err
		}
		if _, err := tw.Write(jsonBytes); err != nil {
			return nil, err
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
	if err := tw.Close(); err != nil {
		return nil, err
	}

	return out, nil
}

func deleteWhiteouts(files map[string]*tarFile, whiteouts []string) map[string]*tarFile {
	for _, whiteout := range whiteouts {
		prefix := strings.Replace(whiteout, ".wh.", "", 1)

		for name, _ := range files {
			if strings.HasPrefix(name, prefix) {
				delete(files, name)
			}
		}
	}

	return files
}

func (e *Export) rewriteChildren(from *ExportedImage) error {
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
			entry = e.ChildOf(entry.LayerConfig.Id)
		} else {
			Debugf("  -  Removing %s. Squashed. (%s)\n", entry.LayerConfig.Id[:12], cmd)

			child := e.ChildOf(entry.LayerConfig.Id)
			if child != nil {
				child.LayerConfig.Parent = entry.LayerConfig.Parent
				delete(e.Entries, entry.LayerConfig.Id)
			}
			entry = child
		}
	}
	return nil
}
