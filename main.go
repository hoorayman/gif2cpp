package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	canvasW        int
	canvasH        int
	threshold      int
	drawMode       string
	scaleMode      string
	outputFormat   string
	varName        string
	outputFile     string
	invertColors   bool
	flipH          bool
	flipV          bool
	rotate         int
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gif2cpp <gif_file>",
		Short: "Convert GIF animations to C/C++ arrays for OLED/LCD displays",
		Long:  "gif2cpp converts GIF animations into C/C++ byte arrays for display on monochrome OLED/LCD screens (ESP32/Arduino).",
		Args:  cobra.ExactArgs(1),
		Run:   runConvert,
	}

	rootCmd.Flags().IntVarP(&canvasW, "width", "W", 128, "Canvas width (pixels)")
	rootCmd.Flags().IntVarP(&canvasH, "height", "H", 64, "Canvas height (pixels)")
	rootCmd.Flags().IntVarP(&threshold, "threshold", "t", 128, "B/W threshold (0-255)")
	rootCmd.Flags().StringVarP(&drawMode, "mode", "m", "horizontal", "Draw mode: horizontal, vertical, horizontal-bytes")
	rootCmd.Flags().StringVarP(&scaleMode, "scale", "s", "fit", "Scale mode: fit, fit-width, fit-height, stretch, custom")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "arduino", "Output format: arduino, plain, esp")
	rootCmd.Flags().StringVarP(&varName, "name", "n", "", "Variable name (default: from filename)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	rootCmd.Flags().BoolVarP(&invertColors, "invert", "i", false, "Invert colors")
	rootCmd.Flags().BoolVar(&flipH, "flip-h", false, "Flip horizontally")
	rootCmd.Flags().BoolVar(&flipV, "flip-v", false, "Flip vertically")
	rootCmd.Flags().IntVar(&rotate, "rotate", 0, "Rotation: 0, 90, 180, 270")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConvert(cmd *cobra.Command, args []string) {
	gifPath := args[0]

	// Validate params
	if threshold < 0 || threshold > 255 {
		fmt.Fprintln(os.Stderr, "Error: threshold must be 0-255")
		os.Exit(1)
	}
	if drawMode != "horizontal" && drawMode != "vertical" && drawMode != "horizontal-bytes" {
		fmt.Fprintln(os.Stderr, "Error: mode must be horizontal, vertical, or horizontal-bytes")
		os.Exit(1)
	}
	if scaleMode != "fit" && scaleMode != "fit-width" && scaleMode != "fit-height" && scaleMode != "stretch" && scaleMode != "custom" {
		fmt.Fprintln(os.Stderr, "Error: scale must be fit, fit-width, fit-height, stretch, or custom")
		os.Exit(1)
	}
	if outputFormat != "arduino" && outputFormat != "plain" && outputFormat != "esp" {
		fmt.Fprintln(os.Stderr, "Error: format must be arduino, plain, or esp")
		os.Exit(1)
	}
	if rotate != 0 && rotate != 90 && rotate != 180 && rotate != 270 {
		fmt.Fprintln(os.Stderr, "Error: rotate must be 0, 90, 180, or 270")
		os.Exit(1)
	}

	// Default variable name from filename
	if varName == "" {
		base := filepath.Base(gifPath)
		ext := filepath.Ext(base)
		varName = sanitizeVarName(strings.TrimSuffix(base, ext))
	}

	// Decode GIF
	gif, err := DecodeGIF(gifPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding GIF: %v\n", err)
		os.Exit(1)
	}

	// Convert frames
	opts := ConvertOptions{
		CanvasWidth:  canvasW,
		CanvasHeight: canvasH,
		Threshold:    uint8(threshold),
		DrawMode:     drawMode,
		ScaleMode:    scaleMode,
		Invert:       invertColors,
		FlipH:        flipH,
		FlipV:        flipV,
		Rotate:       rotate,
	}

	frames, delays, err := ConvertFrames(gif, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting frames: %v\n", err)
		os.Exit(1)
	}

	// Generate output
	output := GenerateOutput(frames, delays, opts, varName, outputFormat)

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Output written to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}
}

func sanitizeVarName(s string) string {
	// Replace non-alphanumeric chars with underscore
	var result strings.Builder
	for i, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			result.WriteRune(r)
		} else {
			if i > 0 {
				result.WriteRune('_')
			}
		}
	}
	name := result.String()
	// Ensure starts with letter or underscore
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	if name == "" {
		name = "gif_data"
	}
	return name
}
