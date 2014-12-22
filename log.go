package libsquash

import (
	"fmt"
	"os"
)

var Verbose bool

func debugf(format string, args ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s", format), args...)
	}
}

func debug(args ...interface{}) {
	if Verbose {
		fmt.Fprintln(os.Stderr, args...)
	}
}
