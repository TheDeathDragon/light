# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library that compiles to Android AAR for controlling RGB LEDs on Android devices. It interfaces with MT6370 PMU LED hardware via sysfs paths.

## Build Commands

```bash
# Build Android AAR (requires gomobile)
gomobile bind -v -o Light.aar -ldflags '-s -w' -target=android -androidapi 30 -target=android/arm64

# Install gomobile (if not installed)
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

## Architecture

**Pattern:** Goroutine-based effect system with context cancellation

**Key Components:**
- `ledcontroller.go` - Main library containing all LED control logic
- Effect system with 17+ predefined effects (bootup, notification, charging, etc.)
- Thread-safe state management with mutex protection
- File handle caching for performance (LED files kept open)

**Concurrency Model:**
- Each effect runs in a separate goroutine
- Context-based cancellation for graceful effect stopping
- `effectDone` channel signals effect completion
- 1-second timeout when stopping active effects

**Public API:**
- `StartEffect(effectType int)` - Start predefined effect
- `StopCurrentEffect()` - Cancel active effect
- `SetRGB(red, green, blue int)` - Direct RGB control (values 0-6)
- `EnableLED(color Color)` - Set solid color
- `TurnOffLED()` - Turn off all LEDs
- `SetLEDEnabled(enabled bool)` / `IsLEDEnabled()` - Master control
- `IsEffectActive()` / `GetCurrentEffect()` - State queries

**Effect Constants:**
```go
EFFECT_NONE = 0
EFFECT_BOOTUP = 1
EFFECT_NOTIFICATION = 2
EFFECT_CALL = 3
EFFECT_CHARGING_LOW/HIGH/COMPLETE = 4-6
EFFECT_WIFI_CONNECTING/CONNECTED/FAILED = 7-9
EFFECT_BLUETOOTH_CONNECTING/CONNECTED/FAILED = 10-12
EFFECT_CAMERA_FOCUS/CAPTURE/SAVE = 13-15
EFFECT_PARTY = 16
EFFECT_MUSIC = 17
```

## Hardware Interface

LED brightness controlled via sysfs (MT6370 PMU):
- Red: `/sys/devices/platform/11016000.i2c5/i2c-5/5-0034/mt6370_pmu_rgbled/leds/mt6370_pmu_led1/brightness`
- Green: `/sys/devices/platform/11016000.i2c5/i2c-5/5-0034/mt6370_pmu_rgbled/leds/mt6370_pmu_led2/brightness`
- Blue: `/sys/devices/platform/11016000.i2c5/i2c-5/5-0034/mt6370_pmu_rgbled/leds/mt6370_pmu_led3/brightness`

Brightness range: 0-6 (MinBrightness to MaxBrightness)

## Debugging

Set `DebugMode = true` to enable debug logging.
