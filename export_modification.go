package libsquash

import (
	"fmt"
	"strings"
	"time"
)

/*
InsertLayer inserts a new layer after "parent" with the token #(squash)
command. Return the new layer
*/
func (e *Export) InsertLayer(parent string) (*Layer, error) {
	id, err := newID()
	if err != nil {
		return nil, err
	}

	layerConfig := NewLayerConfig(id, parent, "squashed w/ libsquash")
	layerConfig.ContainerConfig().Cmd = []string{"/bin/sh", "-c", fmt.Sprintf("#(squash) from %s", parent[:12])}

	entry := &Layer{
		LayerConfig: layerConfig,
	}

	entry.LayerConfig.Created = time.Now().UTC()

	// rewrite child json
	child := e.ChildOf(parent)
	if child != nil {
		child.LayerConfig.Parent = id
	}

	e.Layers[id] = entry

	return entry, err
}

/*
ReplaceLayer refreshes layer "orig" by updating the Created timestamp and ID
and wiring it in correctly
*/
func (e *Export) ReplaceLayer(orig *Layer) error {
	newID, err := newID()
	if err != nil {
		return err
	}

	oldID := orig.LayerConfig.ID
	child := e.ChildOf(oldID)

	newLayer := orig.Clone()

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
