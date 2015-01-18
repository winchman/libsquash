package libsquash

import (
	"encoding/json"

	"github.com/winchman/libsquash/tarball"
)

// helper functions below

func ignore(t *tarball.TarFile) bool {
	if nameRegex.MatchString(t.Name()) {
		return true
	}
	nameParts := t.NameParts()

	if len(nameParts) == 0 {
		return true
	}

	return false
}

func nameFirst(t *tarball.TarFile) string {
	nameParts := t.NameParts()
	if len(nameParts) < 1 {
		return ""
	}

	return nameParts[0]
}

func nameSecond(t *tarball.TarFile) string {
	nameParts := t.NameParts()
	if len(nameParts) < 2 {
		return ""
	}

	return nameParts[1]
}

func repositories(t *tarball.TarFile) bool {
	return nameFirst(t) == "repositories"
}

func jsonFile(t *tarball.TarFile) bool {
	return nameSecond(t) == "json"
}

func layerTar(t *tarball.TarFile) bool {
	return nameSecond(t) == "layer.tar"
}

func versionFile(t *tarball.TarFile) bool {
	return nameSecond(t) == "VERSION"
}

func isDir(t *tarball.TarFile) bool {
	nameParts := t.NameParts()
	return len(nameParts) > 1 && nameParts[1] == ""
}

func checkRepositories(e *export, t *tarball.TarFile) error {
	if err := json.NewDecoder(t.Stream).Decode(&e.Repositories); err != nil {
		return err
	}

	// Export may have multiple branches with the same parent.
	// We can't handle that currently so abort.
	for _, v := range e.Repositories {
		commits := map[string]string{}
		for tag, commit := range *v {
			commits[commit] = tag
		}
		if len(commits) > 1 {
			return errorMultipleBranchesSameParent
		}
	}

	return nil
}
