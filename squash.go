package libsquash

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

var (
	errorMultipleBranchesSameParent = errors.New("this image is a full repository export w/ multiple images in it. " +
		"Please generate the export from a specific image ID or tag.",
	)
	errorNoFROM = errors.New("no layer matching FROM")
)

func Squash(inStream io.Reader) (io.Reader, error) {
	var tarReader = tar.NewReader(inStream)

	var export = newExport()

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
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
				bytes, err := ioutil.ReadAll(tarReader)
				if err != nil {
					return nil, err
				}
				if err = json.Unmarshal(bytes, &export.Repositories); err != nil {
					return nil, err
				}
			}
			continue
		}

		uuidPart := nameParts[0]
		fileName := nameParts[1]
		if export.Entries[uuidPart] == nil {
			export.Entries[uuidPart] = &ExportedImage{}
		}

		switch fileName {
		case "":
			export.Entries[uuidPart].DirHeader = header
		case "json":
			export.Entries[uuidPart].JsonHeader = header
			bytes, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
			if err = json.Unmarshal(bytes, &export.Entries[uuidPart].LayerConfig); err != nil {
				return nil, err
			}
		case "layer.tar":
			export.Entries[uuidPart].LayerTarHeader = header
			_, err := export.Entries[uuidPart].LayerTarBuffer.ReadFrom(tarReader)
			if err != nil {
				return nil, err
			}
		case "VERSION":
			export.Entries[uuidPart].VersionHeader = header
		}
	}

	// Export may have multiple branches with the same parent.
	// We can't handle that currently so abort.
	for _, v := range export.Repositories {
		commits := map[string]string{}
		for tag, commit := range *v {
			commits[commit] = tag
		}
		if len(commits) > 1 {
			return nil, errorMultipleBranchesSameParent
		}
	}

	start := export.FirstSquash()
	// Can't find a previously squashed layer, use first ADD
	if start == nil {
		start = export.FirstFrom()
	}
	// Can't find a FROM, default to root
	if start == nil {
		start = export.Root()
	}

	if start == nil {
		return nil, errorNoFROM
	}

	// insert a new layer after our squash point
	newEntry, err := export.InsertLayer(start.LayerConfig.Id)
	if err != nil {
		return nil, err
	}

	Debugf("Inserted new layer %s after %s\n", newEntry.LayerConfig.Id[0:12], newEntry.LayerConfig.Parent[0:12])

	if Verbose {
		printVerbose(export, newEntry.LayerConfig.Id)
	}

	// squash all later layers into our new layer
	reader, err := export.SquashLayers(newEntry, start)
	//reader, err := export.SquashLayers(newEntry, newEntry)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func printVerbose(export *export, newEntryID string) {
	e := export.Root()
	for {
		if e == nil {
			break
		}
		cmd := strings.Join(e.LayerConfig.ContainerConfig().Cmd, " ")
		if len(cmd) > 60 {
			cmd = cmd[:60]
		}

		if e.LayerConfig.Id == newEntryID {
			Debugf("  -> %s %s\n", e.LayerConfig.Id[0:12], cmd)
		} else {
			Debugf("  -  %s %s\n", e.LayerConfig.Id[0:12], cmd)
		}
		e = export.ChildOf(e.LayerConfig.Id)
	}
}
