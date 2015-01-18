package tarball

import (
	"archive/tar"
	"io"
	"os"
	"strings"
)

type TarFile struct {
	Header *tar.Header
	Stream io.Reader
}

func (t *TarFile) Name() string {
	return t.Header.Name
}

func (t *TarFile) NameParts() []string {
	return strings.Split(t.Header.Name, string(os.PathSeparator))
}
