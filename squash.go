package libsquash

import (
	"errors"
	"io"
	"strings"
)

var (
	errorMultipleBranchesSameParent = errors.New("this image is a full repository export w/ multiple images in it. " +
		"Please generate the export from a specific image ID or tag.",
	)
	errorNoFROM = errors.New("no layer matching FROM")
)

func Squash(instream io.Reader, instream2 io.Reader, outstream io.Writer) (err error) {
	var export = newExport()

	// populate metadata from first stream
	export.parseLayerMetadata(instream)

	// insert a new layer after our squash point
	newEntry, err := export.InsertLayer(export.start.LayerConfig.Id)
	if err != nil {
		return err
	}

	debugf("Inserted new layer %s after %s\n", newEntry.LayerConfig.Id[0:12], newEntry.LayerConfig.Parent[0:12])

	if Verbose {
		printVerbose(export, newEntry.LayerConfig.Id)
	}

	// squash all later layers into our new layer (from second stream)
	return export.SquashLayers(newEntry, export.start, instream2, outstream)
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
			debugf("  -> %s %s\n", e.LayerConfig.Id[0:12], cmd)
		} else {
			debugf("  -  %s %s\n", e.LayerConfig.Id[0:12], cmd)
		}
		e = export.ChildOf(e.LayerConfig.Id)
	}
}
