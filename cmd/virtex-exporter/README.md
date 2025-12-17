# Virtex Exporter

A server that exports EchoVR session data in Virtex format. Supports both live polling from a running EchoVR instance and replay from recorded files.

## Features

- **Live Mode**: Polls `/session` and `/user_bones` endpoints from a running EchoVR instance
- **Replay Mode**: Reads from `.echoreplay` or `.nevrcap` files
- **Virtex Format**: Outputs data in the Virtex format with bone transformations split into XY and ZW components
- **HTTP Server**: Exposes data via a simple HTTP endpoint

## Building

```bash
make virtex-exporter
```

This will create the `virtex-exporter` binary in the current directory.

## Usage

### Live Mode

Poll data from a running EchoVR instance:

```bash
./virtex-exporter -mode live -source 192.168.1.100:6721 -bind 127.0.0.1:8080
```

Options:
- `-source`: The `host:port` of the EchoVR instance (default API port is 6721)
- `-bind`: The address to bind the HTTP server to (default: `127.0.0.1:8080`)
- `-stream-link`: Optional Twitch or other stream URL to include in the response

### Replay Mode

Play back from a recorded file:

```bash
./virtex-exporter -mode replay -source recording.echoreplay -bind 127.0.0.1:8080 -loop
```

Options:
- `-source`: Path to the `.echoreplay` or `.nevrcap` file
- `-bind`: The address to bind the HTTP server to (default: `127.0.0.1:8080`)
- `-loop`: Loop the replay continuously
- `-stream-link`: Optional Twitch or other stream URL to include in the response

## Endpoints

### `GET /`

Returns server information and status in HTML format.

### `GET /stream`

Returns the current session state in Virtex format as JSON:

```json
{
  "Data": {
    "Session": {
      // Session data for players in the stadium
    },
    "Bones": {
      "User_Bones": [
        {
          "Rotation": {"XY": [0, 0], "ZW": [0, 0]},
          "Translation": {"XY": [0, 0], "ZW": [0, 0]},
          "Scale3D": {"XY": [1, 1], "ZW": [1, 1]},
          "Parameters": [0, 0, 0, 0]
        }
        // ... repeats for each bone (typically 22 bones per player)
      ]
    },
    "StreamTimecode": "2025-12-17T10:30:45.123456Z",
    "StreamLink": "https://twitch.tv/..."
  }
}
```

## Data Format

### Bone Structure

Each bone in the `User_Bones` array contains:

- **Rotation**: Quaternion rotation split into XY (x, y) and ZW (z, w)
- **Translation**: 3D position split into XY (x, y) and ZW (z, w with w=padding)
- **Scale3D**: 3D scale split into XY (x, y) and ZW (z, w with w=padding)
- **Parameters**: 4-element array of additional parameters

The bone data comes from EchoVR's `bone_t` (translation) and `bone_o` (orientation/rotation) arrays, which contain 4 floats per bone packed sequentially.

## Examples

### Live streaming with Twitch link

```bash
./virtex-exporter \
  -mode live \
  -source 192.168.1.100:6721 \
  -bind 0.0.0.0:8080 \
  -stream-link "https://twitch.tv/yourstream"
```

### Replay a match recording in a loop

```bash
./virtex-exporter \
  -mode replay \
  -source match_2024-10-20.echoreplay \
  -bind 127.0.0.1:8080 \
  -loop
```

### Query the stream endpoint

```bash
curl http://localhost:8080/stream | jq
```

## Notes

- In live mode, the server polls at 10Hz (every 100ms)
- In replay mode, playback runs at 1x speed (matching the original recording timestamps)
- The server will return HTTP 204 (No Content) if no frame data is available yet
- Session data follows the standard EchoVR API session format
- All bone transformations are split into XY/ZW format as specified by Virtex
