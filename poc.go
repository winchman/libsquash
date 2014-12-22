package main

//import (
//"io"
//"io/ioutil"
//"os"
//"sync"

//"github.com/fsouza/go-dockerclient"
//"github.com/jwilder/docker-squash/libsquash"
//"github.com/rafecolton/go-dockerclient-quick"
//)

//func main() {
////libsquash.Verbose = true
//var wg sync.WaitGroup
//client, err := dockerclient.NewDockerClient()
//if err != nil {
//println(err)
//os.Exit(1)
//}

//imageReader, pipeWriter := io.Pipe()
//defer pipeWriter.Close()
//defer imageReader.Close()

//go func() {
//wg.Add(1)
//imageID := "7159d30ad4cf7e02bdd44fe2611d3db5f412016b38ed55516f562c48d0feb3a3"
//opts := docker.ExportImageOptions{
//Name:         imageID,
//OutputStream: pipeWriter,
//}
//if err := client.Client().ExportImage(opts); err != nil {
//println(err)
//os.Exit(1)
//}
//}()

//tempdir, err := ioutil.TempDir("", "docker-squash")
//if err != nil {
//println(err)
//os.Exit(1)
//}

//defer func() {
//if err := os.RemoveAll(tempdir); err != nil {
//println(err)
//}
//}()

//reader, err := libsquash.Squash(imageReader, tempdir)
//if err != nil {
//println(err)
//os.Exit(1)
//}

//importOpts := docker.ImportImageOptions{
//Source:       "-",
//Repository:   "test",
//Tag:          "foo",
//InputStream:  reader,
//OutputStream: os.Stdout,
//}

//client.Client().ImportImage(importOpts)

//wg.Wait()
//}
