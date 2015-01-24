package libsquash

type tagInfo map[string]string

// An Export contains the layers of the image as well as various forms of
// metadata.  An export "ingests" this metadata and then uses it to determine
// how to compose the squashed image
type Export struct {
	Layers       map[string]*Layer
	Repositories map[string]*tagInfo
	fileToLayers map[string][]fileLoc
	layerToFiles map[string]map[string]bool
	start        *Layer
	whiteouts    []whiteoutFile
}

type fileLoc struct {
	uuid     string
	whiteout bool // to indicate that the file as presented in this layer is a whiteout instead of a regular file
}

type whiteoutFile struct {
	uuid   string // the layer in which the whiteout was found
	prefix string // the file name prefix (without the ".wh." part)
}

// NewExport returns a fully initialized *Export
func NewExport() *Export {
	return &Export{
		Layers:       map[string]*Layer{},
		Repositories: map[string]*tagInfo{},
		fileToLayers: map[string][]fileLoc{},
		layerToFiles: map[string]map[string]bool{},
		whiteouts:    []whiteoutFile{},
	}
}
