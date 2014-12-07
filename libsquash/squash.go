package libsquash

import (
	"errors"
	"io"
	"strings"
)

func Squash(reader io.Reader, tempdir string) (io.Reader, error) {
	export, err := LoadExport(reader, tempdir)
	if err != nil {
		return nil, err
	}

	// Export may have multiple branches with the same parent.
	// We can't handle that currently so abort.
	for _, v := range export.Repositories {
		commits := map[string]string{}
		for tag, commit := range *v {
			commits[commit] = tag
		}
		if len(commits) > 1 {
			return nil, errors.New(
				"This image is a full repository export w/ multiple images in it.  " +
					"You need to generate the export from a specific image ID or tag.",
			)

		}

	}

	start := export.FirstSquash()
	// Can't find a previously squashed layer, use first FROM
	if start == nil {
		start = export.FirstFrom()
	}
	// Can't find a FROM, default to root
	if start == nil {
		start = export.Root()
	}

	if start == nil {
		return nil, errors.New("no layer matching FROM")
	}

	// extract each "layer.tar" to "layer" dir
	err = export.ExtractLayers()
	if err != nil {
		return nil, err
	}

	// insert a new layer after our squash point
	newEntry, err := export.InsertLayer(start.LayerConfig.Id)
	if err != nil {
		return nil, err
	}

	Debugf("Inserted new layer %s after %s\n", newEntry.LayerConfig.Id[0:12],
		newEntry.LayerConfig.Parent[0:12])

	if Verbose {
		e := export.Root()
		for {
			if e == nil {
				break
			}
			cmd := strings.Join(e.LayerConfig.ContainerConfig().Cmd, " ")
			if len(cmd) > 60 {
				cmd = cmd[:60]
			}

			if e.LayerConfig.Id == newEntry.LayerConfig.Id {
				Debugf("  -> %s %s\n", e.LayerConfig.Id[0:12], cmd)
			} else {
				Debugf("  -  %s %s\n", e.LayerConfig.Id[0:12], cmd)
			}
			e = export.ChildOf(e.LayerConfig.Id)
		}
	}

	// squash all later layers into our new layer
	if err = export.SquashLayers(newEntry, newEntry); err != nil {
		return nil, err
	}

	Debugf("Tarring up squashed layer %s\n", newEntry.LayerConfig.Id[:12])
	// create a layer.tar from our squashed layer
	if err = newEntry.TarLayer(); err != nil {
		return nil, err
	}

	Debugf("Removing extracted layers\n")
	// remove our expanded "layer" dirs
	if err = export.RemoveExtractedLayers(); err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()

	// bundle up the new image
	if err = export.TarLayers(writer); err != nil {
		return nil, err
	}

	Debug("Done. New image created.")
	// print our new history
	export.PrintHistory()

	return reader, nil
}
