package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"os"

	xdraw "golang.org/x/image/draw"
)

// DecodeGIF reads and decodes a GIF file
func DecodeGIF(path string) (*gif.GIF, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		return nil, fmt.Errorf("decode gif: %w", err)
	}
	return g, nil
}

// ConvertOptions holds all conversion parameters
type ConvertOptions struct {
	CanvasWidth  int
	CanvasHeight int
	Threshold    uint8
	DrawMode     string // horizontal, vertical, horizontal-bytes
	ScaleMode    string // fit, fit-width, fit-height, stretch, custom
	Invert       bool
	FlipH        bool
	FlipV        bool
	Rotate       int // 0, 90, 180, 270
}

// ConvertFrames processes all GIF frames into monochrome byte arrays
func ConvertFrames(g *gif.GIF, opts ConvertOptions) ([][]byte, []int, error) {
	frames := make([][]byte, 0, len(g.Image))
	delays := make([]int, 0, len(g.Image))

	// Get GIF bounds
	bounds := g.Image[0].Bounds()
	gifW, gifH := bounds.Dx(), bounds.Dy()

	// Calculate scale
	scaledW, scaledH := calcScale(gifW, gifH, opts)

	// Apply rotation to canvas size
	canvasW, canvasH := opts.CanvasWidth, opts.CanvasHeight
	if opts.Rotate == 90 || opts.Rotate == 270 {
		canvasW, canvasH = canvasH, canvasW
	}

	// Process each frame
	var prevImg *image.RGBA

	for i, srcPaletted := range g.Image {
		// Convert paletted to RGBA for consistent processing
		src := image.NewRGBA(srcPaletted.Bounds())
		xdraw.Draw(src, srcPaletted.Bounds(), srcPaletted, srcPaletted.Bounds().Min, xdraw.Src)

		// Handle GIF disposal: compose onto previous frame if needed
		if i > 0 && prevImg != nil {
			composite := image.NewRGBA(prevImg.Bounds())
			xdraw.Draw(composite, prevImg.Bounds(), prevImg, image.Point{}, xdraw.Src)

			if g.Disposal[i-1] != gif.DisposalPrevious {
				xdraw.Draw(composite, srcPaletted.Bounds(), src, srcPaletted.Bounds().Min, xdraw.Over)
			}
			src = composite
		}

		// Scale
		scaled := scaleImage(src, scaledW, scaledH)

		// Place on canvas
		canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
		ox := (canvasW - scaledW) / 2
		oy := (canvasH - scaledH) / 2
		xdraw.Draw(canvas, image.Rect(ox, oy, ox+scaledW, oy+scaledH), scaled, image.Point{}, xdraw.Over)

		// Apply transformations
		result := transformImage(canvas, opts)

		// Convert to monochrome byte array
		frameBytes := imageToBytes(result, opts)

		frames = append(frames, frameBytes)
		// GIF delay is in 100ths of a second, convert to ms
		delayMs := g.Delay[i] * 10
		if delayMs == 0 {
			delayMs = 100 // default 100ms if delay is 0
		}
		delays = append(delays, delayMs)

		prevImg = result
	}

	return frames, delays, nil
}

// calcScale returns the scaled dimensions based on scale mode
func calcScale(gifW, gifH int, opts ConvertOptions) (int, int) {
	canvasW, canvasH := opts.CanvasWidth, opts.CanvasHeight

	switch opts.ScaleMode {
	case "fit":
		scaleX := float64(canvasW) / float64(gifW)
		scaleY := float64(canvasH) / float64(gifH)
		scale := math.Min(scaleX, scaleY)
		return max(1, int(math.Round(float64(gifW)*scale))), max(1, int(math.Round(float64(gifH)*scale)))

	case "fit-width":
		scale := float64(canvasW) / float64(gifW)
		return canvasW, max(1, int(math.Round(float64(gifH)*scale)))

	case "fit-height":
		scale := float64(canvasH) / float64(gifH)
		return max(1, int(math.Round(float64(gifW)*scale))), canvasH

	case "stretch":
		return canvasW, canvasH

	case "custom":
		return canvasW, canvasH

	default:
		return canvasW, canvasH
	}
}

// scaleImage scales an image to the target dimensions
func scaleImage(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// transformImage applies flip and rotation
func transformImage(src *image.RGBA, opts ConvertOptions) *image.RGBA {
	img := src

	// Flip
	if opts.FlipH || opts.FlipV {
		b := img.Bounds()
		dst := image.NewRGBA(b)
		for y := 0; y < b.Dy(); y++ {
			for x := 0; x < b.Dx(); x++ {
				sx, sy := x, y
				if opts.FlipH {
					sx = b.Dx() - 1 - x
				}
				if opts.FlipV {
					sy = b.Dy() - 1 - y
				}
				dst.SetRGBA(x, y, img.RGBAAt(sx, sy))
			}
		}
		img = dst
	}

	// Rotate
	if opts.Rotate != 0 {
		img = rotateImage(img, opts.Rotate)
	}

	return img
}

// rotateImage rotates image by the given degrees (90, 180, 270)
func rotateImage(src *image.RGBA, degrees int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	var dst *image.RGBA
	switch degrees {
	case 90:
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetRGBA(h-1-y, x, src.RGBAAt(x, y))
			}
		}
	case 180:
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetRGBA(w-1-x, h-1-y, src.RGBAAt(x, y))
			}
		}
	case 270:
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dst.SetRGBA(y, w-1-x, src.RGBAAt(x, y))
			}
		}
	default:
		return src
	}
	return dst
}

// imageToBytes converts an RGBA image to a monochrome byte array
func imageToBytes(img *image.RGBA, opts ConvertOptions) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	var result []byte

	switch opts.DrawMode {
	case "horizontal", "horizontal-bytes":
		// Horizontal scan: each row is packed left-to-right, MSB first
		bytesPerRow := (w + 7) / 8
		result = make([]byte, bytesPerRow*h)

		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if isPixelWhite(img, x, y, opts) {
					byteIndex := y*bytesPerRow + x/8
					bitIndex := uint(7 - x%8)
					result[byteIndex] |= 1 << bitIndex
				}
			}
		}

	case "vertical":
		// Vertical scan: each column is packed top-to-bottom, MSB first
		// Pages of 8 pixels vertically
		pages := (h + 7) / 8
		bytesPerCol := pages
		result = make([]byte, w*bytesPerCol)

		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				if isPixelWhite(img, x, y, opts) {
					page := y / 8
					bitInPage := uint(y % 8)
					byteIndex := x*bytesPerCol + page
					result[byteIndex] |= 1 << bitInPage
				}
			}
		}
	}

	return result
}

// isPixelWhite determines if a pixel is "white" (on) based on threshold
func isPixelWhite(img *image.RGBA, x, y int, opts ConvertOptions) bool {
	c := img.RGBAAt(x, y)
	lum := luminance(c.R, c.G, c.B)
	isWhite := lum > opts.Threshold
	if opts.Invert {
		isWhite = !isWhite
	}
	return isWhite
}

// luminance calculates grayscale value from RGB
func luminance(r, g, b uint8) uint8 {
	return uint8(float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114)
}

// max returns the larger of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure color package is used
var _ = color.RGBA{}
