package libsquash

import (
	"bytes"
	//"io/ioutil"
	//"os"
	//"os/exec"
	"archive/tar"
	"time"
)

type ExportedImage struct {
	LayerConfig    *LayerConfig
	LayerTarBuffer bytes.Buffer
	DirHeader      *tar.Header
	VersionHeader  *tar.Header
	JsonHeader     *tar.Header
	LayerTarHeader *tar.Header
}

func newLayerConfig(id, parent, comment string) *LayerConfig {
	return &LayerConfig{
		Id:            id,
		Parent:        parent,
		Comment:       comment,
		Created:       time.Now().UTC(),
		DockerVersion: "0.1.2",
		Architecture:  "x86_64",
	}
}

func (e *ExportedImage) WriteVersion() (err error) {
	//fp := e.VersionPath
	//f, err := os.Create(fp)
	//if err != nil {
	//return err
	//}
	//defer f.Close()

	//_, err = f.WriteString("1.0")
	//if err != nil {
	//return err
	//}

	return err
}

//func (e *ExportedImage) TarLayer() error {

////cwd, err := os.Getwd()
////if err != nil {
////return err
////}

////err = os.Chdir(e.LayerDirPath)
////if err != nil {
////return err
////}
////defer os.Chdir(cwd)

////readCloser, err := archive.Tar(e.LayerDirPath, archive.Uncompressed)
////if err != nil {
////return err
////}

////archiveBytes, err := ioutil.ReadAll(readCloser)
////if err != nil {
////return err
////}
////return ioutil.WriteFile(e.LayerDirPath+"../layer.tar", archiveBytes, 0644)

//cmd := exec.Command("tar", "cvf", "../layer.tar", ".")
//cmd.Dir = e.LayerDirPath
//out, err := cmd.CombinedOutput()
//if err != nil {
//println("GOT HERE 2")
//println(string(out))
//return err
//}
//return nil
//}
