package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rafecolton/libsquash"
)

var (
	buildVersion string
	signals      chan os.Signal
	wg           sync.WaitGroup
)

func shutdown() {
	defer wg.Done()
	<-signals
}

func main() {
	var from, input, output, tag string
	var keepTemp, version bool
	flag.StringVar(&input, "i", "", "Read from a tar archive file, instead of STDIN")
	flag.StringVar(&output, "o", "", "Write to a file, instead of STDOUT")
	flag.StringVar(&tag, "t", "", "Repository name and tag for new image")
	flag.StringVar(&from, "from", "", "Squash from layer ID (default: first FROM layer)")
	flag.BoolVar(&keepTemp, "keepTemp", false, "Keep temp dir when done. (Useful for debugging)")
	flag.BoolVar(&libsquash.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&version, "v", false, "Print version information and quit")

	flag.Usage = func() {
		fmt.Printf("\nUsage: docker-squash [options]\n\n")
		fmt.Printf("Squashes the layers of a tar archive on STDIN and streams it to STDOUT\n\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if version {
		fmt.Println(buildVersion)
		return
	}

	var err error

	if tag != "" && strings.Contains(tag, ":") {
		parts := strings.Split(tag, ":")
		if parts[0] == "" || parts[1] == "" {
			fatalf("bad tag format: %s\n", tag)
		}
	}

	signals = make(chan os.Signal, 1)

	if !keepTemp {
		wg.Add(1)
		signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGTERM)
		go shutdown()
	}

	ir := os.Stdin
	if input != "" {
		var err error
		ir, err = os.Open(input)
		if err != nil {
			fatal(err)
		}
	}

	reader, err := libsquash.Squash(ir)
	if err != nil {
		fatal(err)
	}

	// Export may have multiple branches with the same parent.
	// We can't handle that currently so abort.
	ow := os.Stdout
	if output != "" {
		var err error
		ow, err = os.Create(output)
		if err != nil {
			fatal(err)
		}
		libsquash.Debugf("Tarring new image to %s\n", output)
	} else {
		libsquash.Debugf("Tarring new image to STDOUT\n")
	}

	byteArr, err := ioutil.ReadAll(reader)
	if err != nil {
		fatal(err)
	}

	if _, err = ow.Write(byteArr); err != nil {
		fatal(err)
	}

	libsquash.Debug("Done. New image created.")

	signals <- os.Interrupt
	wg.Wait()
}
