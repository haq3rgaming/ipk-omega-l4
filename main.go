package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := parseArgs(os.Args[1:]) // Parse command-line arguments, returns config struct and error if any
	if err != nil {                    // If any error occurred during argument parsing, print the error and usage
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		os.Exit(1) // Exit with non-zero status to indicate error
	}

	if cfg.help { // If help flag is set, print usage and exit
		printUsage()
		os.Exit(0) // Exit with zero status to indicate success
	}

	if cfg.listIfacesOnly {
		if err := printActiveInterfaces(); err != nil { // If error occurs while listing interfaces, print the error and exit
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1) // Exit with non-zero status to indicate error
		}
		os.Exit(0) // Exit with zero status to indicate success
	}

	if err := runScanner(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1) // Exit with non-zero status to indicate error
	}
}
