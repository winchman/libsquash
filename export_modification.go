package libsquash

import (
	"fmt"
	"strings"
	"time"
)

func (e *export) InsertLayer(parent string) (*layer, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	layerConfig := newLayerConfig(id, parent, "squashed w/ docker-squash")
	layerConfig.ContainerConfig().Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(squash) from %s", parent[:12])}

	entry := &layer{
		LayerConfig: layerConfig,
	}

	entry.LayerConfig.Created = time.Now().UTC()

	// rewrite child json
	child := e.ChildOf(parent)
	child.LayerConfig.Parent = id

	e.Layers[id] = entry

	return entry, err
}

func (e *export) ReplaceLayer(orig *layer) error {
	newID, err := newID()
	if err != nil {
		return err
	}

	oldID := orig.LayerConfig.ID
	child := e.ChildOf(oldID)

	newLayer := &layer{LayerConfig: orig.LayerConfig}
	newLayer.LayerConfig.Created = time.Now().UTC()
	newLayer.LayerConfig.ID = newID

	cmd := strings.Join(orig.LayerConfig.ContainerConfig().Cmd, " ")
	if len(cmd) > 60 {
		cmd = cmd[:60]
	}

	debugf("  -  Replacing %s w/ new layer %s (%s)\n", oldID[:12], newID[:12], cmd)
	if child != nil {
		e.Layers[child.LayerConfig.ID].LayerConfig.Parent = newID
	}

	e.Layers[newID] = newLayer
	delete(e.Layers, oldID)

	return nil
}

func (e *export) RemoveLayer(l *layer) {
	layerID := l.LayerConfig.ID

	debugf("  -  Removing %s. Squashed. (%s)\n", layerID[:12], l.Cmd())

	child := e.ChildOf(layerID)
	if child != nil {
		e.Layers[child.LayerConfig.ID].LayerConfig.Parent = l.LayerConfig.Parent
	}
	delete(e.Layers, layerID)
}
