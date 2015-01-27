package libsquash

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/winchman/libsquash/tarball"
)

/*
SquashLayers produces the #(squash) layer from the contents in the tarball,
rewrites the subsequent layers, using e.RewriteChildren, and then rewrites
the final image tar by calling e.RebuildImage
*/
func (e *Export) SquashLayers(into, from *Layer, tarstream io.Reader, outstream io.Writer) (imageID string, err error) {
	tempfile, err := ioutil.TempFile("", "libsquash")
	if err != nil {
		return "", err
	}

	defer func() {
		_ = tempfile.Close()
		_ = os.RemoveAll(tempfile.Name())
	}()

	var squashLayerTarWriter = tarball.NewTarstream(tempfile)

	// write contents of layer.tar of "squash layer" into tempfile
	if err = tarball.Walk(tarstream, func(t *tarball.TarFile) error {
		nameParts := t.NameParts()
		switch ParseType(t) {
		case Directory:
			uuidPart := nameParts[0]
			e.Layers[uuidPart].DirHeader = t.Header
		case LayerTar:
			uuidPart := nameParts[0]
			e.Layers[uuidPart].LayerTarHeader = t.Header
			if err := tarball.Walk(t.Stream, func(tf *tarball.TarFile) error {
				filePath := nameWithoutWhiteoutPrefix(tf.Name())
				if e.layerToFiles[uuidPart][filePath] {
					if err := squashLayerTarWriter.Add(&tarball.TarFile{Header: tf.Header, Stream: tf.Stream}); err != nil {
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

	// rewind the tempfile so layer.tar for the squash layer can be read
	if _, err = tempfile.Seek(0, 0); err != nil {
		return "", err
	}

	// rewrite the subsequent layers
	debug("  -  Rewriting child history")
	if err := e.RewriteChildren(from, into.LayerConfig.ID); err != nil {
		return "", err
	}

	// rebuild the image tarball for the squashed layer
	return e.RebuildImage(into, outstream, tempfile)
}

/*
RewriteChildren should only be called internally by SquashLayers.

RewriteChildren contains the core logic about how to modify all layers that are
inherited by the squash layer. The logic is as follows:

 - if the layer modifies the filesystem (is a RUN or a #(nop) ADD)
	* remove it
	* that layer will effectively be merged into the squash layer
	* history from these layers will be lost
 - if the layer does NOT modify the filesystem (is any other command type)
	* keep it, but give it a new ID and timestamp
	* the history of that layer and its changes (e.g. new env vars, new workdir, etc.) will be preserved
*/
func (e *Export) RewriteChildren(from *Layer, squashID string) error {
	entry := from
	for {
		if entry == nil {
			break
		}
		child := e.ChildOf(entry.LayerConfig.ID)
		if entry.LayerConfig.ID != squashID {
			if err := e.ReplaceLayer(entry); err != nil {
				return err
			}
		}
		entry = child
	}
	return nil
}
