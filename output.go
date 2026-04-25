package main

import (
	"fmt"
	"strings"
)

func GenerateOutput(frames [][]byte, delays []int, opts ConvertOptions, varName, outputFormat string) string {
	var sb strings.Builder

	frameCount := len(frames)
	canvasW, canvasH := opts.CanvasWidth, opts.CanvasHeight
	if opts.Rotate == 90 || opts.Rotate == 270 {
		canvasW, canvasH = canvasH, canvasW
	}
	bytesPerFrame := len(frames[0])

	sb.WriteString("// GIF2CPP output\n")
	sb.WriteString(fmt.Sprintf("// Frames: %d, Size: %dx%d, Mode: %s\n\n",
		frameCount, canvasW, canvasH, opts.DrawMode))

	headerGuard := strings.ToUpper(varName) + "_H"
	sb.WriteString(fmt.Sprintf("#ifndef %s\n#define %s\n\n", headerGuard, headerGuard))
	sb.WriteString("#include <stdint.h>\n")
	if outputFormat == "arduino" || outputFormat == "esp" {
		sb.WriteString("#include <pgmspace.h>\n")
	}
	sb.WriteString("\n")

	upperName := strings.ToUpper(varName)
	sb.WriteString(fmt.Sprintf("#define %s_FRAME_COUNT %d\n", upperName, frameCount))
	sb.WriteString(fmt.Sprintf("#define %s_WIDTH       %d\n", upperName, canvasW))
	sb.WriteString(fmt.Sprintf("#define %s_HEIGHT      %d\n", upperName, canvasH))
	sb.WriteString(fmt.Sprintf("#define %s_BYTES_PER_FRAME %d\n\n", upperName, bytesPerFrame))

	sb.WriteString(fmt.Sprintf("const uint16_t %s_delays[%d] = {", varName, frameCount))
	for i, d := range delays {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%d", d))
	}
	sb.WriteString("};\n\n")

	var storageAttr string
	switch outputFormat {
	case "arduino":
		storageAttr = "PROGMEM "
	case "esp":
		storageAttr = "PROGMEM "
	default:
		storageAttr = ""
	}

	sb.WriteString(fmt.Sprintf("// %d bytes per frame\n", bytesPerFrame))
	sb.WriteString(fmt.Sprintf("const %suint8_t %s_frames[%d][%d] = {\n",
		storageAttr, varName, frameCount, bytesPerFrame))

	for i, frame := range frames {
		sb.WriteString("  {")
		for j, b := range frame {
			if j%16 == 0 {
				sb.WriteString("\n    ")
			}
			sb.WriteString(fmt.Sprintf("0x%02x,", b))
		}
		sb.WriteString("\n  }")
		if i < frameCount-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("};\n\n")

	// 提供简单的播放宏，不用 struct 避免多头文件冲突
	sb.WriteString(fmt.Sprintf(
		"// Usage:\n// for(int i=0;i<%s_FRAME_COUNT;i++){\n//   display.drawBitmap(0,0,%s_frames[i],%s_WIDTH,%s_HEIGHT,1);\n//   delay(%s_delays[i]);\n// }\n\n",
		upperName, varName, upperName, upperName, varName))

	sb.WriteString(fmt.Sprintf("#endif // %s\n", headerGuard))
	return sb.String()
}
