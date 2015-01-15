package libsquash

type tagInfo map[string]string

type export struct {
	Layers       map[string]*layer
	Repositories map[string]*tagInfo
	fileToLayers map[string][]fileLoc
	layerToFiles map[string]map[string]bool
	start        *layer
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

func newExport() *export {
	return &export{
		Layers:       map[string]*layer{},
		Repositories: map[string]*tagInfo{},
		fileToLayers: map[string][]fileLoc{},
		layerToFiles: map[string]map[string]bool{},
		whiteouts:    []whiteoutFile{},
	}
}
