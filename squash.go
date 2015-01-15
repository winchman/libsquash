package libsquash

import (
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

/*
Squash squashes a docker image where instream is an io.Reader for the image
tarball, outstream is an io.Writer to which the squashed image tarball will be written,
and imageIDOut is an io.Writer to which the id of the squashed image will be written
*/
func Squash(instream io.Reader, outstream io.Writer, imageIDOut io.Writer) error {
	export := newExport()
	tempfile, err := ioutil.TempFile("", "libsquash")
	if err != nil {
		return err
	}

	defer func() {
		_ = tempfile.Close()
		_ = os.RemoveAll(tempfile.Name())
	}()

	instreamTee := io.TeeReader(instream, tempfile)

	// populate metadata from first stream
	export.parseLayerMetadata(instreamTee)

	// rewind tempfile to the entire tar stream can be read back in
	if _, err = tempfile.Seek(0, 0); err != nil {
		return err
	}

	// insert a new layer after our squash point
	newEntry, err := export.InsertLayer(export.start.LayerConfig.ID)
	if err != nil {
		return err
	}

	debugf("Inserted new layer %s after %s\n", newEntry.LayerConfig.ID[0:12], newEntry.LayerConfig.Parent[0:12])

	if Verbose {
		printVerbose(export, newEntry.LayerConfig.ID)
	}

	// squash all later layers into our new layer (from second stream)
	imageID, err := export.SquashLayers(newEntry, export.start, tempfile, outstream)
	if err != nil {
		return err
	}

	if _, err := imageIDOut.Write([]byte(imageID)); err != nil {
		return err
	}

	return nil
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

		if e.LayerConfig.ID == newEntryID {
			debugf("  -> %s %s\n", e.LayerConfig.ID[0:12], cmd)
		} else {
			debugf("  -  %s %s\n", e.LayerConfig.ID[0:12], cmd)
		}
		e = export.ChildOf(e.LayerConfig.ID)
	}
}
