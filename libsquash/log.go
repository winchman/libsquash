package libsquash

import (
	"fmt"
	"os"
)

var Verbose bool

func Debugf(format string, args ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s", format), args...)
	}
}

func Debug(args ...interface{}) {
	if Verbose {
		fmt.Fprintln(os.Stderr, args...)
	}
}
