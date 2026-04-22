# GIF2CPP

A command-line tool written in Go that converts GIF animations into C/C++ byte arrays for display on monochrome OLED/LCD screens (ESP32, Arduino, STM32, etc.).

Inspired by the [online GIF2CPP tool](https://huykhong.com/IOT/gif2cpp/), but runs entirely locally — no browser needed.

> **gif2cpp** | gif to c++ | gif to cpp | gif to arduino | gif to esp32 | gif animation to oled | gif to ssd1306 | gif to byte array | animated gif c++ | image2cpp alternative | oled animation converter | monochrome display gif

## Features

- Decode multi-frame GIF animations into monochrome bitmaps
- Configurable B/W threshold for optimal image quality
- Three draw modes: `horizontal`, `vertical`, `horizontal-bytes`
- Five scale modes: `fit`, `fit-width`, `fit-height`, `stretch`, `custom`
- Flip horizontally/vertically and rotate (0°/90°/180°/270°)
- Color inversion support
- Three output formats: Arduino (PROGMEM), plain C/C++, ESP32/ESP8266 (ICACHE_RODATA_ATTR)
- Generates `.h` header files with `AnimatedGIF` struct, ready to use

## Install

```bash
go install github.com/hoorayman/gif2cpp@latest
```

Or build from source:

```bash
git clone https://github.com/hoorayman/gif2cpp.git
cd gif2cpp
go build -o gif2cpp .
```

## Usage

```bash
gif2cpp <gif_file> [flags]
```

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--width` | `-W` | `128` | Canvas width in pixels |
| `--height` | `-H` | `64` | Canvas height in pixels |
| `--threshold` | `-t` | `128` | B/W threshold (0–255) |
| `--mode` | `-m` | `horizontal` | Draw mode: `horizontal`, `vertical`, `horizontal-bytes` |
| `--scale` | `-s` | `fit` | Scale mode: `fit`, `fit-width`, `fit-height`, `stretch`, `custom` |
| `--format` | `-f` | `arduino` | Output format: `arduino`, `plain`, `esp` |
| `--name` | `-n` | *(from filename)* | Variable name in generated code |
| `--output` | `-o` | *(stdout)* | Output file path |
| `--invert` | `-i` | `false` | Invert colors |
| `--flip-h` | | `false` | Flip horizontally |
| `--flip-v` | | `false` | Flip vertically |
| `--rotate` | | `0` | Rotation: `0`, `90`, `180`, `270` |

### Examples

```bash
# Basic usage — output to stdout
gif2cpp animation.gif

# Generate Arduino PROGMEM header file
gif2cpp cat.gif -o cat.h

# ESP32 format with inverted colors and custom threshold
gif2cpp image.gif -o image.h -f esp -i -t 100

# Vertical scan mode, rotated 90°, fit to height
gif2cpp demo.gif -m vertical --rotate 90 -s fit-height -o demo.h

# Custom variable name and plain C output
gif2cpp logo.gif -n myLogo -f plain -o logo.h
```

## Output Format

The tool generates a `.h` header file containing:

1. **`AnimatedGIF` struct** — universal struct for all GIF animations
2. **Frame delay array** — per-frame timing in milliseconds
3. **Frame data array** — monochrome bitmap bytes (PROGMEM/const)
4. **`AnimatedGIF` instance** — ready to use with `playGIF()`

Example output:

```c
#ifndef CAT_H
#define CAT_H

#include <stdint.h>
#include <pgmspace.h>

typedef struct AnimatedGIF {
    const uint8_t frame_count;
    const uint16_t width;
    const uint16_t height;
    const uint16_t* delays;
    const uint8_t (* frames)[1024];
} AnimatedGIF;

#define CAT_FRAME_COUNT 12
#define CAT_WIDTH 128
#define CAT_HEIGHT 64

const uint16_t cat_delays[CAT_FRAME_COUNT] = {100, 100, 100, ...};

PROGMEM const uint8_t cat_frames[CAT_FRAME_COUNT][1024] = {
  { 0x00, 0x00, 0x03, 0xf8, ... },
  { 0x00, 0x00, 0x07, 0xfc, ... },
  // ...
};

const AnimatedGIF cat_gif = {
    .frame_count = CAT_FRAME_COUNT,
    .width = CAT_WIDTH,
    .height = CAT_HEIGHT,
    .delays = cat_delays,
    .frames = cat_frames
};

// Usage: playGIF(&cat_gif);

#endif // CAT_H
```

## Arduino Integration

Use the generated `.h` file with the `playGIF()` function:

```cpp
#include <Adafruit_SSD1306.h>
#include <Wire.h>
#include "cat.h"

Adafruit_SSD1306 display(128, 64, &Wire, -1);

void playGIF(const AnimatedGIF* gif, uint16_t loopCount = 1) {
  for (uint16_t loop = 0; loop < loopCount; loop++) {
    for (uint8_t frame = 0; frame < gif->frame_count; frame++) {
      display.clearDisplay();
      for (uint16_t y = 0; y < gif->height; y++) {
        for (uint16_t x = 0; x < gif->width; x++) {
          uint16_t byteIndex = y * (((gif->width + 7) / 8)) + (x / 8);
          uint8_t bitIndex = 7 - (x % 8);
          if (gif->frames[frame][byteIndex] & (1 << bitIndex)) {
            display.drawPixel(x, y, WHITE);
          }
        }
      }
      display.display();
      delay(gif->delays[frame]);
    }
  }
}

void setup() {
  Wire.begin(21, 22);
  display.begin(SSD1306_SWITCHCAPVCC, 0x3C);
  playGIF(&cat_gif);
}

void loop() {}
```

## Output Formats

| Format | Keyword | Description |
|---|---|---|
| Arduino | `arduino` | `PROGMEM const uint8_t` arrays |
| Plain C/C++ | `plain` | `const uint8_t` arrays (no PROGMEM) |
| ESP32/ESP8266 | `esp` | `ICACHE_RODATA_ATTR PROGMEM const uint8_t` arrays |

## Draw Modes

| Mode | Description |
|---|---|
| `horizontal` | Scan horizontal, MSB first (default, for most OLEDs) |
| `vertical` | Scan vertical, MSB first (for some LCD controllers) |
| `horizontal-bytes` | Scan horizontal byte-by-byte |

## License

MIT License
