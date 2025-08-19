# Echo Replay Converter

A command-line tool to convert `.echoreplay` files to `.nevrcap` format.

## Overview

This tool reads compressed `.echoreplay` files (ZIP archives containing frame data) and converts them to structured `.nevrcap` JSON files. It provides robust options for bulk processing with safety features.

## Installation

Build the application:

```bash
go build -o echoreplay-converter ./cmd/echoreplay-converter
```

## Usage

```bash
./echoreplay-converter [options] <glob-pattern>
```

### Arguments

- `glob-pattern`: Glob pattern to match `.echoreplay` files (e.g., `"*.echoreplay"` or `"data/*.echoreplay"`)

### Options

- `--dry-run`: Simulate the conversion process without making any changes
- `--remove-original`: Remove the original `.echoreplay` files after successful conversion
- `--verbose`: Enable verbose output showing detailed progress
- `--version`: Show version information

### Examples

```bash
# Convert all .echoreplay files in current directory
./echoreplay-converter "*.echoreplay"

# Simulate conversion without making changes
./echoreplay-converter --dry-run "data/*.echoreplay"

# Convert files and remove originals
./echoreplay-converter --remove-original "*.echoreplay"

# Convert with verbose output
./echoreplay-converter --verbose "recordings/**/*.echoreplay"
```

## File Formats

### Input Format (.echoreplay)

- ZIP archive containing frame data
- Each line format: `timestamp\tsession_data\tbone_data\n`
- Timestamp format: `2006/01/02 15:04:05.000`
- Session data and bone data are typically JSON

### Output Format (.nevrcap)

JSON file with structured data:

```json
{
  "version": "1.0",
  "created_at": "2024-01-01T12:00:00Z",
  "source_file": "original.echoreplay",
  "frames": [
    {
      "timestamp": "2024-01-01T12:00:00Z",
      "session_data": { /* parsed JSON or string */ },
      "player_bone_data": { /* parsed JSON or string */ }
    }
  ]
}
```

## Error Handling

The tool provides comprehensive error handling:

- Validates input file existence and format
- Reports conversion progress and failures
- Gracefully handles malformed data
- Provides clear error messages for troubleshooting

## Safety Features

- **Dry Run**: Test conversions without making changes
- **Verbose Mode**: Detailed progress and error reporting
- **Original File Removal**: Only removes files after successful conversion
- **Pattern Matching**: Processes only files matching the specified pattern

## Testing

Run the test suite:

```bash
go test ./converter -v
```

The tests create mock `.echoreplay` files and verify the complete conversion pipeline.