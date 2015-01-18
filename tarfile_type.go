package libsquash

import (
	"regexp"

	"github.com/winchman/libsquash/tarball"
)

type LayerFileType uint8

const (
	IGNORE LayerFileType = iota

	REPOSITORIES

	DIRECTORY

	JSON

	LAYER_TAR

	VERSION

	UNKNOWN
)

var nameRegex = regexp.MustCompile(`^\.$|^\.\.$|^\.\/$`)

func Type(t *tarball.TarFile) LayerFileType {
	if nameRegex.MatchString(t.Name()) {
		return IGNORE
	}
	nameParts := t.NameParts()

	if len(nameParts) == 0 {
		return IGNORE
	} else if len(nameParts) == 1 {
		if nameParts[0] == "repositories" {
			return REPOSITORIES
		} else {
			return UNKNOWN
		}
	} else if len(nameParts) == 2 {
		switch nameParts[1] {
		case "":
			return DIRECTORY
		case "json":
			return JSON
		case "layer.tar":
			return LAYER_TAR
		case "VERSION":
			return VERSION
		}
	}

	return UNKNOWN
}
