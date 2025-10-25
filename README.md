# Waybar weather

My simple waybar weather module. Heavily based, even copied, from [`weather-script`](https://github.com/flipflop133/weather-script) but in Go.

## Usage

```json
"custom/weather": {
    "exec": "waybar-weather",
    "exec-if": "command -v waybar-weather &>/dev/null",
    "interval": 600
    "return-type": "json",
}
```

or

```bash
waybar-weather --settings path/to/weather-settings.json
```

The module assumes the `weather-icons.json` is in the same path as the settings.

### Required settings

- url
- parameters
- key
