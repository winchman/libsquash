package libsquash

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
)

/*
Squash squashes a docker image where instream is an io.Reader for the image
tarball, outstream is an io.Writer to which the squashed image tarball will be written,
and imageIDOut is an io.Writer to which the id of the squashed image will be written

The steps are as follows:

1. Go through stream, tee'ing it to a tempfile, get layer configs and layer->file lists

2. Using the metadata, go through the tar stream again (from the tempfile),
build the squash layer, build the final image tar, and write it to our output stream

3. (as a cleanup step, write the id of the final layer, which the daemon will
use as the image id)
*/
func Squash(instream io.Reader, outstream io.Writer, imageIDOut io.Writer) error {
	export := NewExport()
	tempfile, err := ioutil.TempFile("", "libsquash")
	if err != nil {
		return err
	}

	defer func() {
		_ = tempfile.Close()
		_ = os.RemoveAll(tempfile.Name())
	}()

	instreamTee := io.TeeReader(instream, tempfile)

	/*
		1. Ingest Image Metadata: populate metadata from first stream
	*/
	export.IngestImageMetadata(instreamTee)

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

	/*
		2. squash all later layers into our new layer (from second stream)
	*/
	imageID, err := export.SquashLayers(newEntry, export.start, tempfile, outstream)
	if err != nil {
		return err
	}

	/*
		3. write the imageID to the imageID output stream
	*/
	if _, err := imageIDOut.Write([]byte(imageID)); err != nil {
		return err
	}

	return nil
}

func printVerbose(export *Export, newEntryID string) {
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
