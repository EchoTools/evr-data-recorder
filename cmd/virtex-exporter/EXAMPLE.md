# Example Virtex Exporter Response

This is an example of the JSON response format returned by the `/stream` endpoint:

```json
{
  "Data": {
    "Session": {
      "sessionid": "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
      "game_status": "in_lobby",
      "match_type": "Echo_Arena",
      "map_name": "mpl_arena_a",
      "private_match": false,
      "blue_team": {
        "score": 0,
        "possession": false,
        "players": [
          {
            "name": "PlayerOne",
            "playerid": 1,
            "level": 50,
            "position": [0, 0, 0],
            "velocity": [0, 0, 0],
            "stunned": false,
            "ping": 25
          }
        ]
      },
      "orange_team": {
        "score": 0,
        "possession": false,
        "players": []
      },
      "disc": {
        "position": [0, 0, 0],
        "velocity": [0, 0, 0]
      },
      "game_clock": 0,
      "game_clock_display": "00:00.00"
    },
    "Bones": {
      "User_Bones": [
        {
          "Rotation": {
            "XY": [0.0, 0.0],
            "ZW": [0.0, 1.0]
          },
          "Translation": {
            "XY": [0.0, 1.5],
            "ZW": [0.0, 0.0]
          },
          "Scale3D": {
            "XY": [1.0, 1.0],
            "ZW": [1.0, 1.0]
          },
          "Parameters": [0.0, 0.0, 0.0, 0.0]
        },
        {
          "Rotation": {
            "XY": [0.0, 0.0],
            "ZW": [0.0, 1.0]
          },
          "Translation": {
            "XY": [0.2, 1.4],
            "ZW": [0.1, 0.0]
          },
          "Scale3D": {
            "XY": [1.0, 1.0],
            "ZW": [1.0, 1.0]
          },
          "Parameters": [0.0, 0.0, 0.0, 0.0]
        }
        // ... continues for all bones (typically 22 per player)
      ]
    },
    "StreamTimecode": "2025-12-17T10:30:45.123456789Z",
    "StreamLink": "https://twitch.tv/yourstream"
  }
}
```

## Field Descriptions

### Session
Contains the full EchoVR session data including:
- Player positions, velocities, and stats
- Team scores and possession
- Disc position and velocity
- Game clock and status
- Match configuration

### Bones
Contains skeletal bone data for all players in the match:

#### User_Bones Array
An array of bone transformations. Each player typically has 22 bones representing:
- Head
- Chest/torso
- Arms (shoulders, elbows, hands)
- Legs (hips, knees, feet)

#### Bone Structure
Each bone contains:
- **Rotation**: Quaternion (x, y, z, w) representing bone orientation
  - XY: First two components (x, y)
  - ZW: Last two components (z, w)
- **Translation**: 3D position (x, y, z) + padding
  - XY: Position x and y components
  - ZW: Position z component and padding
- **Scale3D**: 3D scale (x, y, z) + padding (typically all 1.0)
  - XY: Scale x and y components
  - ZW: Scale z component and padding
- **Parameters**: Four additional float parameters (typically zeros)

### StreamTimecode
ISO 8601 timestamp indicating when this frame was captured or played back.

### StreamLink
Optional URL to the stream (e.g., Twitch, YouTube) if provided via `-stream-link` flag.
