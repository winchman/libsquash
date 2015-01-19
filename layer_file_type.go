package libsquash

import (
	"regexp"

	"github.com/winchman/libsquash/tarball"
)

// LayerFileType is a type for identifying a file in a tarball as specific to
// how this library parses a docker image
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

// ParseType returns the LayerFileType of the given tar file
func ParseType(t *tarball.TarFile) LayerFileType {
	if LayerFileIgnoreRegex.MatchString(t.Name()) {
		return Ignore
	}
	nameParts := t.NameParts()
	switch len(nameParts) {
	case 0:
		return Ignore
	case 1:
		if nameParts[0] == "repositories" {
			return Repositories
		}
		return Unknown
	case 2:
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
