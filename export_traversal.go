package libsquash

import (
	"fmt"
	"strings"
)

func (e *Export) firstLayer(pattern string) *Layer {
	root := e.Root()
	for {
		if root == nil {
			break
		}
		cmd := strings.Join(root.LayerConfig.ContainerConfig().Cmd, " ")
		if strings.Contains(cmd, pattern) {
			break
		}
		root = e.ChildOf(root.LayerConfig.ID)
	}
	return root
}

// FirstSquash finds the first layer marked with the token #(squash)
func (e *Export) FirstSquash() *Layer {
	return e.firstLayer("#(squash)")
}

// Root returns the top layer in the export
func (e *Export) Root() *Layer {
	return e.ChildOf("")
}

// Last returns the layer found last in the list
func (e *Export) Last() *Layer {
	current := e.Root()
	for {
		if current == nil {
			break
		}
		child := e.ChildOf(current.LayerConfig.ID)
		if child == nil {
			break
		}
		current = child
	}
	return current
}

// ChildOf returns the child layer or nil of the parent
func (e *Export) ChildOf(parent string) *Layer {
	for _, entry := range e.Layers {
		if entry.LayerConfig.Parent == parent {
			return entry
		}
	}
	return nil
}

// GetByID returns an exportedImaged with a prefix matching ID.  An error
// is returned multiple exportedImages matched.
func (e *Export) GetByID(idPrefix string) (*Layer, error) {
	matches := []*Layer{}
	for id, entry := range e.Layers {
		if strings.HasPrefix(id, idPrefix) {
			matches = append(matches, entry)
		}
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("%s is ambiguous - %d matched", idPrefix, len(matches))
	}

	if len(matches) == 0 {
		return nil, nil
	}

	return matches[0], nil
}
