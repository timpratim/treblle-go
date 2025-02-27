package main

import (
	"flag"
	"fmt"

	"github.com/timpratim/treblle-go/internal"
)

func main() {
	// Define CLI flags
	debug := flag.Bool("debug", false, "Show Treblle SDK debug information")

	// Parse command-line arguments
	flag.Parse()

	// If `-debug` flag is passed, display debug information
	if *debug {
		internal.DebugCommand()
	} else {
		fmt.Println("Usage: treblle-go -debug")
		fmt.Println("\nTreblle Go SDK CLI")
		fmt.Println("-------------------")
		fmt.Println("Available commands:")
		fmt.Println("  -debug    Show SDK configuration information")
	}
}
