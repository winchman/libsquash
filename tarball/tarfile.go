package tarball

import (
	"archive/tar"
	"io"
	"os"
	"strings"
)

// TarFile is a representation of a file in a tarball. It consists of two parts
// - the header and the stream. The header is a regular tar header, the stream
// is a byte stream that can be used to read the file's contents
type TarFile struct {
	Header *tar.Header
	Stream io.Reader
}

// Name is a convenience function for returning the name of the underlying file
func (t *TarFile) Name() string {
	return t.Header.Name
}

// NameParts is a convenience function for returning an array of the parts of
// the file's name, split by os.PathSeparator
func (t *TarFile) NameParts() []string {
	return strings.Split(t.Header.Name, string(os.PathSeparator))
}
