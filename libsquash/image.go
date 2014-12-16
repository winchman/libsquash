package libsquash

import (
	"encoding/json"
	//"io/ioutil"
	"os"
	"os/exec"
	"time"

	//"github.com/docker/docker/pkg/archive"
)

type ExportedImage struct {
	Path         string
	JsonPath     string
	VersionPath  string
	LayerTarPath string
	LayerDirPath string
	LayerConfig  *LayerConfig
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

func (e *ExportedImage) WriteVersion() error {
	fp := e.VersionPath
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString("1.0")
	if err != nil {
		return err
	}

	return err
}

func (e *ExportedImage) WriteJson() error {
	fp := e.JsonPath
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	jb, err := json.Marshal(e.LayerConfig)
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(jb))
	if err != nil {
		return err
	}

	return err
}

func (e *ExportedImage) CreateDirs() error {
	return os.MkdirAll(e.Path, 0755)
}

func (e *ExportedImage) TarLayer() error {

	//cwd, err := os.Getwd()
	//if err != nil {
		//return err
	//}

	//err = os.Chdir(e.LayerDirPath)
	//if err != nil {
		//return err
	//}
	//defer os.Chdir(cwd)

	//readCloser, err := archive.Tar(e.LayerDirPath, archive.Uncompressed)
	//if err != nil {
	//return err
	//}

	//archiveBytes, err := ioutil.ReadAll(readCloser)
	//if err != nil {
	//return err
	//}
	//return ioutil.WriteFile(e.LayerDirPath+"../layer.tar", archiveBytes, 0644)

	cmd := exec.Command("tar", "cvf", "../layer.tar", ".")
	cmd.Dir = e.LayerDirPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		println("GOT HERE 2")
		println(string(out))
		return err
	}
	return nil
}

func (e *ExportedImage) RemoveLayerDir() error {
	return os.RemoveAll(e.LayerDirPath)
}

func (e *ExportedImage) ExtractLayerDir() error {
	err := os.MkdirAll(e.LayerDirPath, 0755)
	if err != nil {
		return err
	}

	out, err := extractTar(e.LayerTarPath, e.LayerDirPath)
	if err != nil {
		println(string(out))
		return err
	}
	return nil
}
