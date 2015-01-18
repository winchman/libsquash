package libsquash

import (
	"regexp"

	"github.com/winchman/libsquash/tarball"
)

// LayerFileType is a type for identifying a file in a tarball as specific to a
// docker imag
type LayerFileType uint8

const (
	// Ignore is for ".", "..", and "./" (identified by LayerFileIgnoreRegex)
	Ignore LayerFileType = iota

	// Repositories is for "repositories"
	Repositories

	// Directory is for "<uuid>/"
	Directory

	// JSON is for "<uuid>/json"
	JSON

	// LayerTar is for "<uuid>/layer.tar"
	LayerTar

	// Version is for "<uuid>/VERSION"
	Version

	// Unknown is for files that cannot be otherwise identified
	Unknown
)

// LayerFileIgnoreRegex is the regex for identifying files that should be of
// type Ignore
var LayerFileIgnoreRegex = regexp.MustCompile(`^\.$|^\.\.$|^\.\/$`)

// Type returns the LayerFileType of the given tarball
func Type(t *tarball.TarFile) LayerFileType {
	if LayerFileIgnoreRegex.MatchString(t.Name()) {
		return Ignore
	}
	nameParts := t.NameParts()

	if len(nameParts) == 0 {
		return Ignore
	} else if len(nameParts) == 1 {
		if nameParts[0] == "repositories" {
			return Repositories
		}
		return Unknown
	} else if len(nameParts) == 2 {
		switch nameParts[1] {
		case "":
			return Directory
		case "json":
			return JSON
		case "layer.tar":
			return LayerTar
		case "VERSION":
			return Version
		}
	}

	return Unknown
}
