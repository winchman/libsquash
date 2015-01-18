package tarball

import (
	"archive/tar"
	"io"
)

// Tarstream is a type for writing tarballs
type Tarstream interface {
	// Close closes the underlying tar writer
	Close() error

	// Add adds tf to the underlying tar writer. First, the header is written.
	// Then, if tf.Stream is not nil, its contents are copied into underlying
	// tar writer
	Add(tf *TarFile) error
}

// NewTarstream returns a tar stream that wries Add()'ed files to outstream
func NewTarstream(outstream io.Writer) Tarstream {
	return &tarstream{
		writer: tar.NewWriter(outstream),
	}
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
