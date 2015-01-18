package tarball

import (
	"archive/tar"
	"io"
)

type Tarstream interface {
	Close() error
	Add(tf *TarFile) error
}

type tarstream struct {
	writer *tar.Writer
}

func (t *tarstream) Close() error {
	return t.writer.Close()
}

func (t *tarstream) Add(tf *TarFile) (err error) {
	if err = t.writer.WriteHeader(tf.Header); err != nil {
		return
	}
	if tf.Stream != nil {
		_, err = io.Copy(t.writer, tf.Stream)
	}
	return
}

func NewTarstream(outstream io.Writer) Tarstream {
	return &tarstream{
		writer: tar.NewWriter(outstream),
	}
}
