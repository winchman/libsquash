package libsquash

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/pkg/units"
)

var uuidRegex = regexp.MustCompile("^[a-f0-9]{64}$")

type TagInfo map[string]string

type Export struct {
	Entries      map[string]*ExportedImage
	Repositories map[string]*TagInfo
	Path         string
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

// LoadExport loads a tarball export created by docker save.
func LoadExport(image io.Reader, tmpdir string) (*Export, error) {
	export := &Export{
		Entries:      map[string]*ExportedImage{},
		Repositories: map[string]*TagInfo{},
		Path:         tmpdir,
	}

	// populate the data structure without writing any files to disk
	if err := export.Populate(image); err != nil {
		return nil, err
	}

	// write the files to disk - TODO: remove this eventually
	if err := export.Extract(image); err != nil {
		return nil, err
	}

	return export, nil
}

func (e *Export) Populate(r io.Reader) error {
	t := tar.NewReader(r)
ReadFromTar:
	for {
		header, err := t.Next()
		if err != nil {
			if err == io.EOF {
				break ReadFromTar
			}
			return err
		}

		if header.Name == "." || header.Name == ".." || header.Name == "./" {
			continue
		}

		// if is json file
		nameParts := strings.Split(header.Name, string(os.PathSeparator))
		if len(nameParts) != 2 {
			continue
		}

		uuidPart := nameParts[0]
		fileName := nameParts[len(nameParts)-1]

		if len(nameParts) == 2 && uuidRegex.MatchString(uuidPart) {
			if fileName == "json" {
				bytes, err := ioutil.ReadAll(t)
				if err != nil {
					return err
				}
				e.Entries[uuidPart] = &ExportedImage{
					Path:         filepath.Join(e.Path, uuidPart),
					JsonPath:     filepath.Join(e.Path, uuidPart, "json"),
					VersionPath:  filepath.Join(e.Path, uuidPart, "VERSION"),
					LayerTarPath: filepath.Join(e.Path, uuidPart, "layer.tar"),
					LayerDirPath: filepath.Join(e.Path, uuidPart, "layer"),
				}
				if err = json.Unmarshal(bytes, &e.Entries[uuidPart].LayerConfig); err != nil {
					return err
				}
			} else if fileName == "repositories" {
				bytes, err := ioutil.ReadAll(t)
				if err != nil {
					return err
				}
				if err = json.Unmarshal(bytes, &e.Repositories); err != nil {
					return err
				}
			}
		}
	}

	Debugf("Loaded image w/ %s layers\n", strconv.FormatInt(int64(len(e.Entries)), 10))
	for repo, tags := range e.Repositories {
		Debugf("  -  %s (%s tags)\n", repo, strconv.FormatInt(int64(len(*tags)), 10))
	}
	return nil
}

func (e *Export) Extract(r io.Reader) error {
	if err := os.MkdirAll(e.Path, 0755); err != nil {
		return err
	}

	t := tar.NewReader(r)
	for {
		header, err := t.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if header.Name == "." || header.Name == ".." || header.Name == "./" {
			continue
		}
		fn := path.Join(e.Path, header.Name)

		if header.FileInfo().IsDir() {
			err = os.Mkdir(fn, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			err := os.Chtimes(fn, time.Now().UTC(), header.FileInfo().ModTime())
			if err != nil {
				return err
			}

			continue
		}

		item, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, header.FileInfo().Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(item, t); err != nil {
			log.Fatalln(err)
		}
		item.Close()
		err = os.Chtimes(fn, time.Now().UTC(), header.FileInfo().ModTime())
		if err != nil {
			return err
		}
	}
}

func (e *Export) ExtractLayers() error {
	Debug("Extracting layers...")

	for _, entry := range e.Entries {
		Debugf("  -  %s\n", entry.LayerTarPath)
		err := entry.ExtractLayerDir()
		if err != nil {
			return err
		}
	}
	return nil
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

func (e *Export) PrintHistory() {
	current := e.Root()
	order := []*ExportedImage{}
	for {
		order = append(order, current)
		current = e.ChildOf(current.LayerConfig.Id)
		if current == nil {
			break
		}
	}

	for i := 0; i < len(order); i++ {
		stat, err := os.Stat(order[i].LayerTarPath)
		size := int64(-1)
		if stat != nil && err == nil {
			size = stat.Size()
		}

		cmd := strings.Join(order[i].LayerConfig.ContainerConfig().Cmd, " ")
		if len(cmd) > 60 {
			cmd = cmd[0:57] + "..."
		}

		Debug("  - ", order[i].LayerConfig.Id[0:12],
			humanDuration(time.Now().UTC().Sub(order[i].LayerConfig.Created.UTC())),
			cmd, units.HumanSize(size))
	}
}

func (e *Export) InsertLayer(parent string) (*ExportedImage, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	layerConfig := newLayerConfig(id, parent, "squashed w/ docker-squash")
	layerConfig.ContainerConfig().Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(squash) from %s", parent[:12])}
	entry := &ExportedImage{
		Path:         filepath.Join(e.Path, id),
		JsonPath:     filepath.Join(e.Path, id, "json"),
		VersionPath:  filepath.Join(e.Path, id, "VERSION"),
		LayerTarPath: filepath.Join(e.Path, id, "layer.tar"),
		LayerDirPath: filepath.Join(e.Path, id, "layer"),
		LayerConfig:  layerConfig,
	}
	entry.LayerConfig.Created = time.Now().UTC()

	err = entry.CreateDirs()
	if err != nil {
		return nil, err
	}

	err = entry.WriteJson()
	if err != nil {
		return nil, err
	}

	err = entry.WriteVersion()
	if err != nil {
		return nil, err
	}

	child := e.ChildOf(parent)
	child.LayerConfig.Parent = id

	err = child.WriteJson()
	if err != nil {
		return nil, err
	}

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
	if len(cmd) > 50 {
		cmd = cmd[:47] + "..."
	}

	Debugf("  -  Replacing %s w/ new layer %s (%s)\n", oldId[:12], id[:12], cmd)
	if child != nil {
		child.LayerConfig.Parent = id
		err = child.WriteJson()
		if err != nil {
			return nil, err
		}
	}

	location := path.Dir(orig.Path)
	layerDir := filepath.Join(location, id)
	err = os.MkdirAll(layerDir, 0755)
	if err != nil {
		return nil, err
	}

	layerConfig := orig.LayerConfig
	layerConfig.Id = id

	entry := &ExportedImage{
		Path:         filepath.Join(location, id),
		JsonPath:     filepath.Join(location, id, "json"),
		VersionPath:  filepath.Join(location, id, "VERSION"),
		LayerTarPath: filepath.Join(location, id, "layer.tar"),
		LayerDirPath: filepath.Join(location, id, "layer"),
		LayerConfig:  layerConfig,
	}
	entry.LayerConfig.Created = time.Now().UTC()

	err = entry.WriteJson()
	if err != nil {
		return nil, err
	}

	e.Entries[id] = entry

	os.Rename(orig.LayerDirPath, entry.LayerDirPath)
	os.Rename(orig.LayerTarPath, entry.LayerTarPath)
	os.Rename(orig.VersionPath, entry.VersionPath)

	err = os.RemoveAll(orig.Path)
	if err != nil {
		return nil, err
	}

	delete(e.Entries, oldId)

	return entry, err
}

func (e *Export) SquashLayers(to, from *ExportedImage) error {

	Debugf("Squashing from %s into %s\n", from.LayerConfig.Id[:12], to.LayerConfig.Id[:12])
	layerDir := filepath.Join(to.Path, "layer")
	err := os.MkdirAll(layerDir, 0755)
	if err != nil {
		return err
	}

	current := from
	if current == nil {
		return errors.New(fmt.Sprintf("%s does not exists", from.LayerConfig.Id))
	}

	order := []*ExportedImage{}
	for {
		order = append(order, current)
		current = e.ChildOf(current.LayerConfig.Id)
		if current == nil {
			break
		}
	}

	for _, entry := range order {
		if _, err := os.Stat(entry.LayerTarPath); os.IsNotExist(err) {
			continue
		}

		out, err := extractTar(entry.LayerTarPath, layerDir)
		if err != nil {
			println(string(out))
			return err
		}
	}
	Debug("  -  Deleting whiteouts")
	err = e.deleteWhiteouts(layerDir)
	if err != nil {
		return err
	}

	Debug("  -  Rewriting child history")
	return e.rewriteChildren(from)
}

func (e *Export) TarLayers(w io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var location string
	for _, entry := range e.Entries {
		location = path.Dir(entry.Path)
		break
	}

	err = os.Chdir(location)
	if err != nil {
		return err
	}
	defer os.Chdir(cwd)

	cmd := exec.Command("tar", "cOf", "-", "*")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		println("GOT HERE 1")
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, err = io.Copy(w, stdout)
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stderr, stderr)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func (e *Export) RemoveExtractedLayers() error {
	for _, entry := range e.Entries {

		err := entry.RemoveLayerDir()
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Export) rewriteChildren(entry *ExportedImage) error {

	squashId := entry.LayerConfig.Id
	for {
		if entry == nil {
			break
		}

		cmd := strings.Join(entry.LayerConfig.ContainerConfig().Cmd, " ")
		if len(cmd) > 50 {
			cmd = cmd[:47] + "..."
		}

		if entry.LayerConfig.Id == squashId {
			entry = e.ChildOf(entry.LayerConfig.Id)
			continue
		}

		if strings.Contains(cmd, "#(nop)") && !strings.Contains(cmd, "ADD") {
			newEntry, err := e.ReplaceLayer(entry.LayerConfig.Id)
			if err != nil {
				return err
			}

			entry = e.ChildOf(newEntry.LayerConfig.Id)
		} else {
			Debugf("  -  Removing %s. Squashed. (%s)\n", entry.LayerConfig.Id[:12], cmd)
			err := os.RemoveAll(entry.Path)
			if err != nil {
				return err
			}

			child := e.ChildOf(entry.LayerConfig.Id)
			delete(e.Entries, entry.LayerConfig.Id)

			if child == nil {
				break
			}

			child.LayerConfig.Parent = entry.LayerConfig.Parent

			err = child.WriteJson()
			if err != nil {
				return err
			}
			entry = child

		}

	}
	return nil
}

func (e *Export) deleteWhiteouts(location string) error {
	return filepath.Walk(location, func(p string, info os.FileInfo, err error) error {
		//fmt.Printf("processing: p => %s, info => %#v, err => %#v\n", p, info, err)
		if err != nil && !os.IsNotExist(err) {
			println("GOT HERE: " + err.Error())
			return err
		}

		if info == nil {
			return nil
		}

		name := info.Name()
		parent := filepath.Dir(p)
		// if start with whiteout
		if strings.Index(name, ".wh.") == 0 {
			deletedFile := path.Join(parent, name[len(".wh."):len(name)])
			// remove deleted files
			if err := os.RemoveAll(deletedFile); err != nil {
				return err
			}
			// remove the whiteout itself
			if err := os.RemoveAll(p); err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *Export) WriteRepositoriesJson() error {
	fp := filepath.Join(e.Path, "repositories")
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	jb, err := json.Marshal(e.Repositories)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(jb))
	if err != nil {
		return err
	}

	return err
}
