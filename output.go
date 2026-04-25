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
	bytesPerFrame := len(frames[0])

	upperName := strings.ToUpper(varName)
	headerGuard := upperName + "_H"

	// Header comment
	sb.WriteString("// GIF2CPP output\n")
	sb.WriteString("// Converted from GIF to C/C++ array for OLED/LCD display\n")
	sb.WriteString(fmt.Sprintf("// Frames: %d, Size: %dx%d pixels, Mode: %s\n",
		frameCount, canvasW, canvasH, opts.DrawMode))
	sb.WriteString(fmt.Sprintf("// Threshold: %d, Scale: %s, Dither: %v\n\n",
		opts.Threshold, opts.ScaleMode, opts.Dither))

	// Include guard
	sb.WriteString(fmt.Sprintf("#ifndef %s\n", headerGuard))
	sb.WriteString(fmt.Sprintf("#define %s\n\n", headerGuard))

	sb.WriteString("#include <stdint.h>\n")
	if outputFormat == "arduino" || outputFormat == "esp" {
		sb.WriteString("#include <pgmspace.h>\n")
	}
	sb.WriteString("\n")

	// Defines
	sb.WriteString(fmt.Sprintf("#define %s_FRAME_COUNT    %d\n", upperName, frameCount))
	sb.WriteString(fmt.Sprintf("#define %s_WIDTH          %d\n", upperName, canvasW))
	sb.WriteString(fmt.Sprintf("#define %s_HEIGHT         %d\n", upperName, canvasH))
	sb.WriteString(fmt.Sprintf("#define %s_BYTES_PER_FRAME %d\n\n", upperName, bytesPerFrame))

	// Delays array
	sb.WriteString(fmt.Sprintf("const uint16_t %s_delays[%s_FRAME_COUNT] = {", varName, upperName))
	for i, d := range delays {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%d", d))
	}
	sb.WriteString("};\n\n")

	// Storage attribute
	var storageAttr string
	switch outputFormat {
	case "arduino", "esp":
		storageAttr = "PROGMEM "
	default:
		storageAttr = ""
	}

	// Frame data array
	sb.WriteString(fmt.Sprintf("// Frame data: %d frames x %d bytes\n", frameCount, bytesPerFrame))
	sb.WriteString(fmt.Sprintf("const %suint8_t %s_frames[%s_FRAME_COUNT][%s_BYTES_PER_FRAME] = {\n",
		storageAttr, varName, upperName, upperName))

	for i, frame := range frames {
		sb.WriteString("  {")
		for j, byt := range frame {
			if j%16 == 0 {
				sb.WriteString("\n    ")
			}
			sb.WriteString(fmt.Sprintf("0x%02x,", byt))
		}
		sb.WriteString("\n  }")
		if i < frameCount-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("};\n\n")

	// Usage example
	sb.WriteString("/*\n")
	sb.WriteString(" * Usage example (Adafruit SSD1306):\n")
	sb.WriteString(" *\n")
	sb.WriteString(fmt.Sprintf(" *   for (int i = 0; i < %s_FRAME_COUNT; i++) {\n", upperName))
	sb.WriteString(" *     display.clearDisplay();\n")
	if outputFormat == "arduino" || outputFormat == "esp" {
		sb.WriteString(fmt.Sprintf(" *     display.drawBitmap(0, 0, %s_frames[i], %s_WIDTH, %s_HEIGHT, WHITE);\n",
			varName, upperName, upperName))
	} else {
		sb.WriteString(fmt.Sprintf(" *     display.drawBitmap(0, 0, %s_frames[i], %s_WIDTH, %s_HEIGHT, WHITE);\n",
			varName, upperName, upperName))
	}
	sb.WriteString(" *     display.display();\n")
	sb.WriteString(fmt.Sprintf(" *     delay(%s_delays[i]);\n", varName))
	sb.WriteString(" *   }\n")
	sb.WriteString(" */\n\n")

	sb.WriteString(fmt.Sprintf("#endif // %s\n", headerGuard))

	return sb.String()
}
