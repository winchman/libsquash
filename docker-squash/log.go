package main

import (
	"fmt"
	"os"
)

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("ERROR: %s", format), args...)
	signals <- os.Interrupt
	wg.Wait()
	os.Exit(1)
}

func fatal(args ...interface{}) {
	fmt.Fprint(os.Stderr, "ERROR: ")
	fmt.Fprintln(os.Stderr, args...)
	signals <- os.Interrupt
	wg.Wait()
	os.Exit(1)
}
