package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/echotools/evr-data-recorder/v3/converter"
)

var version = "v1.0.0"

func main() {
	var (
		removeOriginal = flag.Bool("remove-original", false, "Remove the original .echoreplay files after conversion")
		dryRun         = flag.Bool("dry-run", false, "Simulate the conversion process without making any changes")
		verbose        = flag.Bool("verbose", false, "Enable verbose output")
		showVersion    = flag.Bool("version", false, "Show version information")
	)

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <glob-pattern>\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nConvert .echoreplay files to .nevrcap format\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Version: %s\n\n", version)
		fmt.Fprintf(flag.CommandLine.Output(), "Arguments:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  glob-pattern    Glob pattern to match .echoreplay files (e.g., \"*.echoreplay\" or \"data/*.echoreplay\")\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %s \"*.echoreplay\"                    # Convert all .echoreplay files in current directory\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s --dry-run \"data/*.echoreplay\"     # Simulate conversion of files in data directory\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s --remove-original \"*.echoreplay\" # Convert files and remove originals\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  %s --verbose \"**/*.echoreplay\"       # Convert with verbose output (recursive)\n", os.Args[0])
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("echoreplay-converter %s\n", version)
		os.Exit(0)
	}

	// Check if glob pattern is provided
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	globPattern := flag.Arg(0)

	// Create options
	options := converter.ConvertOptions{
		RemoveOriginal: *removeOriginal,
		DryRun:         *dryRun,
		Verbose:        *verbose,
	}

	// Perform conversion
	if err := converter.ConvertFiles(globPattern, options); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}