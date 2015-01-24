package tarball

import (
	"archive/tar"
	"io"
)

// WalkFunc is a func for handling each file (header and byte stream) in a tarball
type WalkFunc func(t *TarFile) error

// Walk walks through the files in the tarball represented by tarstream and
// passes each of them to the WalkFunc provided as an argument
func Walk(tarstream io.Reader, walkFunc WalkFunc) error {
	reader := tar.NewReader(tarstream)
ReadLoop:
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break ReadLoop
			}
			return err
		}
		if err := walkFunc(&TarFile{Header: header, Stream: reader}); err != nil {
			return err
		}
	}
	return nil
}
