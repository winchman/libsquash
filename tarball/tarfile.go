package tarball

import (
	"archive/tar"
	"io"
	"os"
	"strings"
)

// TarFile is a representation of a file in a tarball. It consists of two parts,
// the Header and the Stream. The Header is a regular tar header, the Stream
// is a byte stream that can be used to read the file's contents
type TarFile struct {
	Header *tar.Header
	Stream io.Reader
}

// Name returns the name of the file as reported by the header
func (t *TarFile) Name() string {
	return t.Header.Name
}

// NameParts returns an array of the parts of the file's name, split by
// os.PathSeparator
func (t *TarFile) NameParts() []string {
	return strings.Split(t.Header.Name, string(os.PathSeparator))
}
