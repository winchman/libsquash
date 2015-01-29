package libsquash

import (
	"regexp"

	"github.com/winchman/libsquash/tarball"
)

// LayerFileType is a type for identifying a file in a tarball as specific to
// how this library parses a docker image
type LayerFileType uint8

const (
	// IgnoreType is for ".", "..", and "./" (identified by LayerFileIgnoreRegex)
	IgnoreType LayerFileType = iota

	// RepositoriesType is for "repositories"
	RepositoriesType

	// DirectoryType is for "<uuid>/"
	DirectoryType

	// JSONType is for "<uuid>/json"
	JSONType

	// LayerTarType is for "<uuid>/layer.tar"
	LayerTarType

	// VersionType is for "<uuid>/VERSION"
	VersionType

	// UnknownType is for files that cannot be otherwise identified
	UnknownType
)

// LayerFileIgnoreRegex is the regex for identifying files that should be of
// type Ignore
var LayerFileIgnoreRegex = regexp.MustCompile(`^\.$|^\.\.$|^\.\/$`)

// ParseType returns the LayerFileType of the given tar file
func ParseType(t *tarball.TarFile) LayerFileType {
	if LayerFileIgnoreRegex.MatchString(t.Name()) {
		return IgnoreType
	}
	nameParts := t.NameParts()
	switch len(nameParts) {
	case 0:
		return IgnoreType
	case 1:
		if nameParts[0] == "repositories" {
			return RepositoriesType
		}
		return UnknownType
	case 2:
		switch nameParts[1] {
		case "":
			return DirectoryType
		case "json":
			return JSONType
		case "layer.tar":
			return LayerTarType
		case "VERSION":
			return VersionType
		}
	}
	return UnknownType
}
