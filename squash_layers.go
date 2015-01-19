package libsquash

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/winchman/libsquash/tarball"
)

func (e *export) SquashLayers(into, from *layer, tarstream io.Reader, outstream io.Writer) (imageID string, err error) {
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
		switch Type(t) {
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
	if err := e.rewriteChildren(into); err != nil {
		return "", err
	}

	// rebuild the image tarball for the squashed layer
	return e.RebuildImage(into, outstream, tempfile)
}

func (e *export) rewriteChildren(from *layer) error {
	var entry = from

	squashID := entry.LayerConfig.ID
	for {
		if entry == nil {
			break
		}
		child := e.ChildOf(entry.LayerConfig.ID)

		// if the layer is not the squash layer
		// => if: we have a #(nop) that is not an ADD, skip it
		// => else: remove the stuff in the layer.tar
		if entry.LayerConfig.ID != squashID && !strings.Contains(entry.Cmd(), "#(squash)") {
			if strings.Contains(entry.Cmd(), "#(nop)") && !strings.Contains(entry.Cmd(), "ADD") {
				if err := e.ReplaceLayer(entry); err != nil {
					return err
				}
			} else {
				e.RemoveLayer(entry)
			}
		}
		entry = child
	}
	return nil
}
