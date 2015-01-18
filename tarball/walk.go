package tarball

import (
	"archive/tar"
	"io"
)

type WalkFunc func(t *TarFile) error

func Walk(tarstream io.Reader, funk WalkFunc) error {
	reader := tar.NewReader(tarstream)

Read:
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break Read
			}
			return err
		}

		file := &TarFile{
			Header: header,
			Stream: reader,
		}

		if err = funk(file); err != nil {
			return err
		}
	}

	return nil
}
