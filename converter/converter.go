package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConvertOptions holds the conversion options
type ConvertOptions struct {
	RemoveOriginal bool
	DryRun         bool
	Verbose        bool
}

// ConvertFile converts a single .echoreplay file to .nevrcap format
func ConvertFile(inputPath string, options ConvertOptions) error {
	if options.Verbose {
		fmt.Printf("Processing: %s\n", inputPath)
	}

	if options.DryRun {
		fmt.Printf("[DRY RUN] Would convert: %s\n", inputPath)
		return nil
	}

	// Generate output filename
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".nevrcap"

	// Read the input file
	reader, err := NewEchoReplayReader(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", inputPath, err)
	}
	defer reader.Close()

	// Create the output writer
	writer := NewNEVRWriter(outputPath, inputPath)

	// Convert frames
	frames, err := reader.ReadAllFrames()
	if err != nil {
		return fmt.Errorf("failed to read frames from %s: %w", inputPath, err)
	}

	for _, frame := range frames {
		if err := writer.WriteFrame(frame); err != nil {
			return fmt.Errorf("failed to write frame: %w", err)
		}
	}

	// Write the output file
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close output file %s: %w", outputPath, err)
	}

	if options.Verbose {
		fmt.Printf("Converted %d frames from %s to %s\n", writer.FrameCount(), inputPath, outputPath)
	}

	// Remove original file if requested
	if options.RemoveOriginal {
		if err := os.Remove(inputPath); err != nil {
			return fmt.Errorf("failed to remove original file %s: %w", inputPath, err)
		}
		if options.Verbose {
			fmt.Printf("Removed original file: %s\n", inputPath)
		}
	}

	return nil
}

// ConvertFiles converts multiple files matching the glob pattern
func ConvertFiles(globPattern string, options ConvertOptions) error {
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return fmt.Errorf("failed to match glob pattern %s: %w", globPattern, err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no files found matching pattern: %s", globPattern)
	}

	var errors []string
	successCount := 0

	for _, match := range matches {
		// Check if it's a regular file (not a directory)
		info, err := os.Stat(match)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to stat %s: %v", match, err))
			continue
		}
		if info.IsDir() {
			continue
		}

		// Only process files with .echoreplay extension
		if !strings.HasSuffix(strings.ToLower(match), ".echoreplay") {
			if options.Verbose {
				fmt.Printf("Skipping non-echoreplay file: %s\n", match)
			}
			continue
		}

		if err := ConvertFile(match, options); err != nil {
			errors = append(errors, fmt.Sprintf("failed to convert %s: %v", match, err))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		fmt.Printf("Conversion completed with %d successes and %d errors:\n", successCount, len(errors))
		for _, errMsg := range errors {
			fmt.Printf("  ERROR: %s\n", errMsg)
		}
		return fmt.Errorf("conversion completed with %d errors", len(errors))
	}

	fmt.Printf("Successfully converted %d files\n", successCount)
	return nil
}