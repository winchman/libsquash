package libsquash

import (
	"fmt"
	"strings"
)

func (e *export) firstLayer(pattern string) *layer {
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

func (e *export) FirstFrom() *layer {
	return e.firstLayer("#(nop) ADD file")
}

func (e *export) FirstSquash() *layer {
	return e.firstLayer("#(squash)")
}

// Root returns the top layer in the export
func (e *export) Root() *layer {
	return e.ChildOf("")
}

func (e *export) LastChild() *layer {
	c := e.Root()
	for {
		if e.ChildOf(c.LayerConfig.ID) == nil {
			break
		}
		c = e.ChildOf(c.LayerConfig.ID)
	}
	return c
}

// ChildOf returns the child layer or nil of the parent
func (e *export) ChildOf(parent string) *layer {
	for _, entry := range e.Layers {
		if entry.LayerConfig.Parent == parent {
			return entry
		}
	}
	return nil
}

// GetById returns an exportedImaged with a prefix matching ID.  An error
// is returned multiple exportedImages matched.
func (e *export) GetByID(idPrefix string) (*layer, error) {
	matches := []*layer{}
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
