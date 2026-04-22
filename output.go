package main

import (
	"fmt"
	"strings"
)

// GenerateOutput produces the C/C++ header file content
func GenerateOutput(frames [][]byte, delays []int, opts ConvertOptions, varName, outputFormat string) string {
	var sb strings.Builder

	frameCount := len(frames)
	canvasW, canvasH := opts.CanvasWidth, opts.CanvasHeight
	if opts.Rotate == 90 || opts.Rotate == 270 {
		canvasW, canvasH = canvasH, canvasW
	}

	// Calculate bytes per frame
	bytesPerFrame := len(frames[0])

	// Header comment
	sb.WriteString("// GIF2CPP output\n")
	sb.WriteString("// Converted from GIF to C/C++ array for OLED/LCD display\n")
	sb.WriteString(fmt.Sprintf("// Frames: %d, Size: %dx%d pixels, Mode: %s\n",
		frameCount, canvasW, canvasH, opts.DrawMode))
	sb.WriteString(fmt.Sprintf("// Threshold: %d, Scale: %s\n",
		opts.Threshold, opts.ScaleMode))
	sb.WriteString("\n")

	// Include guards and headers
	headerGuard := strings.ToUpper(varName) + "_H"
	sb.WriteString(fmt.Sprintf("#ifndef %s\n", headerGuard))
	sb.WriteString(fmt.Sprintf("#define %s\n\n", headerGuard))

	sb.WriteString("#include <stdint.h>\n")
	if outputFormat == "arduino" || outputFormat == "esp" {
		sb.WriteString("#include <pgmspace.h>\n")
	}
	sb.WriteString("\n")

	// AnimatedGIF struct definition (only if not already defined)
	sb.WriteString("#ifndef ANIMATED_GIF_DEFINED\n")
	sb.WriteString("#define ANIMATED_GIF_DEFINED\n")
	sb.WriteString("typedef struct AnimatedGIF {\n")
	sb.WriteString("    const uint8_t frame_count;\n")
	sb.WriteString("    const uint16_t width;\n")
	sb.WriteString("    const uint16_t height;\n")
	sb.WriteString("    const uint16_t* delays;\n")
	sb.WriteString(fmt.Sprintf("    const uint8_t (* frames)[%d];\n", bytesPerFrame))
	sb.WriteString("} AnimatedGIF;\n")
	sb.WriteString("#endif\n\n")

	// Defines
	upperName := strings.ToUpper(varName)
	sb.WriteString(fmt.Sprintf("#define %s_FRAME_COUNT %d\n", upperName, frameCount))
	sb.WriteString(fmt.Sprintf("#define %s_WIDTH %d\n", upperName, canvasW))
	sb.WriteString(fmt.Sprintf("#define %s_HEIGHT %d\n\n", upperName, canvasH))

	// Delays array
	sb.WriteString(fmt.Sprintf("const uint16_t %s_delays[%s_FRAME_COUNT] = {", varName, upperName))
	for i, d := range delays {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%d", d))
	}
	sb.WriteString("};\n\n")

	// Frame data array
	var storageAttr string
	switch outputFormat {
	case "arduino":
		storageAttr = "PROGMEM const "
	case "esp":
		storageAttr = "ICACHE_RODATA_ATTR PROGMEM const "
	case "plain":
		storageAttr = "const "
	}

	sb.WriteString(fmt.Sprintf("// Frame data - %d bytes per frame\n", bytesPerFrame))
	sb.WriteString(fmt.Sprintf("%suint8_t %s_frames[%s_FRAME_COUNT][%d] = {\n",
		storageAttr, varName, upperName, bytesPerFrame))

	for i, frame := range frames {
		sb.WriteString("  {\n")
		sb.WriteString("    ")
		for j, b := range frame {
			if j > 0 {
				if j%16 == 0 {
					sb.WriteString("\n    ")
				} else {
					sb.WriteString(" ")
				}
			}
			sb.WriteString(fmt.Sprintf("0x%02x,", b))
		}
		sb.WriteString("\n  }")
		if i < len(frames)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("};\n\n")

	// AnimatedGIF instance
	sb.WriteString(fmt.Sprintf("const AnimatedGIF %s_gif = {\n", varName))
	sb.WriteString(fmt.Sprintf("    .frame_count = %s_FRAME_COUNT,\n", upperName))
	sb.WriteString(fmt.Sprintf("    .width = %s_WIDTH,\n", upperName))
	sb.WriteString(fmt.Sprintf("    .height = %s_HEIGHT,\n", upperName))
	sb.WriteString(fmt.Sprintf("    .delays = %s_delays,\n", varName))
	sb.WriteString(fmt.Sprintf("    .frames = %s_frames\n", varName))
	sb.WriteString("};\n\n")

	// Usage comment
	sb.WriteString("// Usage: playGIF(&")
	sb.WriteString(varName)
	sb.WriteString("_gif);\n\n")

	sb.WriteString(fmt.Sprintf("#endif // %s\n", headerGuard))

	return sb.String()
}
