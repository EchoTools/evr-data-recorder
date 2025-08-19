# evr-data-recorder

Session and player bone data recorder for Echo VR.

## Components

### Main Recorder (`main.go`)
Records session and player bone data from Echo VR servers.

### Echo Replay Converter (`cmd/echoreplay-converter`)
Converts `.echoreplay` files to `.nevrcap` format for analysis and processing.

## Building

Build all components:
```bash
make all
```

Build individual components:
```bash
make build      # Main recorder
make converter  # Replay converter
```

## Usage

### Data Recorder
```bash
./bin/evr-data-recorder [options] host:port
```

### Replay Converter
```bash
./bin/echoreplay-converter [options] <glob-pattern>
```

See [docs/echoreplay-converter.md](docs/echoreplay-converter.md) for detailed converter documentation.

## Testing

```bash
make test
```
