# Examples

## bouncing_ball

一个小球绕圆运动的 16 帧动画，适合 128×64 OLED 屏幕。

### 文件

| 文件 | 说明 |
|---|---|
| `bouncing_ball.gif` | 原始 GIF 动画（64×64，16帧） |
| `bouncing_ball.h` | 生成的 ESP32 C++ 头文件 |

### 生成命令

```bash
# ESP32 格式（ICACHE_RODATA_ATTR + PROGMEM），适配 128×64 屏幕
gif2cpp examples/bouncing_ball.gif -f esp -n bouncing_ball -W 128 -H 64 -o examples/bouncing_ball.h
```

### Arduino/ESP32 使用

```cpp
#include <Wire.h>
#include <Adafruit_SSD1306.h>
#include "bouncing_ball.h"

#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 64
Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, -1);

void playGIF(const AnimatedGIF* gif) {
  static unsigned long lastFrame = 0;
  static uint16_t currentFrame = 0;
  unsigned long now = millis();

  if (now - lastFrame >= gif->delays[currentFrame]) {
    lastFrame = now;
    display.clearDisplay();

    const uint8_t* frame = gif->frames[currentFrame];
    for (int y = 0; y < gif->height; y++) {
      for (int x = 0; x < gif->width; x++) {
        int byteIndex = (y * gif->width + x) / 8;
        int bitIndex = 7 - (x % 8);
        if (frame[byteIndex] & (1 << bitIndex)) {
          display.drawPixel(x, y, SSD1306_WHITE);
        }
      }
    }
    display.display();
    currentFrame = (currentFrame + 1) % gif->frame_count;
  }
}

void setup() {
  Wire.begin(21, 22);
  display.begin(SSD1306_SWITCHCAPVCC, 0x3C);
}

void loop() {
  playGIF(&bouncing_ball_gif);
}
```

### 其他格式示例

```bash
# Arduino 格式（仅 PROGMEM）
gif2cpp examples/bouncing_ball.gif -f arduino -n bouncing_ball -W 128 -H 64 -o bouncing_ball.h

# 纯 C 格式（无 PROGMEM，适合模拟器或 PC）
gif2cpp examples/bouncing_ball.gif -f plain -n bouncing_ball -W 128 -H 64 -o bouncing_ball.h

# 反色 + 自定义阈值
gif2cpp examples/bouncing_ball.gif -f esp -n bouncing_ball_inv -W 128 -H 64 -i -t 80 -o bouncing_ball_inv.h
```
