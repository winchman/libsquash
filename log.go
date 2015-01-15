package libsquash

import (
	"fmt"
	"os"
)

// Verbose will print out debugging info when set to true. Should be set
// manually in code only for debugging purposes
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
