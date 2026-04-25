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
	Rotate       int  // 0, 90, 180, 270
	Dither       bool // Enable Floyd-Steinberg dithering
}

// ConvertFrames processes all GIF frames into monochrome byte arrays
func ConvertFrames(g *gif.GIF, opts ConvertOptions) ([][]byte, []int, error) {
	frames := make([][]byte, 0, len(g.Image))
	delays := make([]int, 0, len(g.Image))

	bounds := g.Image[0].Bounds()
	gifW, gifH := bounds.Dx(), bounds.Dy()
	scaledW, scaledH := calcScale(gifW, gifH, opts)

	canvasW, canvasH := opts.CanvasWidth, opts.CanvasHeight
	if opts.Rotate == 90 || opts.Rotate == 270 {
		canvasW, canvasH = canvasH, canvasW
	}

	// Use full GIF logical size as composite canvas
	gifBounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	if gifBounds.Dx() == 0 || gifBounds.Dy() == 0 {
		gifBounds = bounds
	}
	composite := image.NewRGBA(gifBounds)

	for i, srcPaletted := range g.Image {
		frameBounds := srcPaletted.Bounds()

		// Handle disposal from previous frame
		if i > 0 && int(i-1) < len(g.Disposal) {
			switch g.Disposal[i-1] {
			case gif.DisposalBackground:
				prev := g.Image[i-1].Bounds()
				for y := prev.Min.Y; y < prev.Max.Y; y++ {
					for x := prev.Min.X; x < prev.Max.X; x++ {
						composite.SetRGBA(x, y, color.RGBA{})
					}
				}
			case gif.DisposalPrevious:
				// keep composite as-is
			}
		}

		// Draw current frame onto composite
		src := image.NewRGBA(frameBounds)
		xdraw.Draw(src, frameBounds, srcPaletted, frameBounds.Min, xdraw.Src)
		xdraw.Draw(composite, frameBounds, src, image.Point{}, xdraw.Over)

		// Scale composite to target size
		scaled := scaleImage(composite, scaledW, scaledH)

		// Place on canvas centered
		canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
		ox := (canvasW - scaledW) / 2
		oy := (canvasH - scaledH) / 2
		xdraw.Draw(canvas, image.Rect(ox, oy, ox+scaledW, oy+scaledH), scaled, image.Point{}, xdraw.Over)

		// Apply flip/rotate
		result := transformImage(canvas, opts)

		// Convert to monochrome bytes
		frameBytes := imageToBytes(result, opts)
		frames = append(frames, frameBytes)

		delayMs := g.Delay[i] * 10
		if delayMs == 0 {
			delayMs = 100
		}
		delays = append(delays, delayMs)
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

// scaleImage scales an image using high-quality CatmullRom interpolation
func scaleImage(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// transformImage applies flip and rotation
func transformImage(src *image.RGBA, opts ConvertOptions) *image.RGBA {
	img := src

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

// imageToBytes converts an RGBA image to a monochrome byte array.
// When Dither is enabled, Floyd-Steinberg error diffusion is applied
// to simulate grayscale on a 1-bit display.
func imageToBytes(img *image.RGBA, opts ConvertOptions) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	// Build grayscale float buffer (0-255), respecting alpha
	gray := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(x, y)
			if c.A < 128 {
				gray[y*w+x] = 0
			} else {
				gray[y*w+x] = float64(luminance(c.R, c.G, c.B))
			}
		}
	}

	// Floyd-Steinberg dithering
	// darkCutoff: pixels below this snap to black without error diffusion.
	// Raised to 60 to aggressively suppress background noise.
	// brightCutoff: pixels above this snap to white without diffusion.
	const darkCutoff = 60.0
	const brightCutoff = 220.0

	if opts.Dither {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				old := gray[y*w+x]
				if old > 255 {
					old = 255
				} else if old < 0 {
					old = 0
				}

				var newVal float64
				var errVal float64

				if old < darkCutoff {
					gray[y*w+x] = 0
					continue
				} else if old > brightCutoff {
					gray[y*w+x] = 255
					continue
				} else {
					if old > 127 {
						newVal = 255
					} else {
						newVal = 0
					}
					gray[y*w+x] = newVal
					errVal = old - newVal
				}

				if x+1 < w {
					gray[y*w+x+1] += errVal * 7 / 16
				}
				if x-1 >= 0 && y+1 < h {
					gray[(y+1)*w+x-1] += errVal * 3 / 16
				}
				if y+1 < h {
					gray[(y+1)*w+x] += errVal * 5 / 16
				}
				if x+1 < w && y+1 < h {
					gray[(y+1)*w+x+1] += errVal * 1 / 16
				}
			}
		}

		// Post-processing: remove isolated white pixels (noise dots).
		// Any white pixel with fewer than 2 white 8-neighbours is killed.
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if gray[y*w+x] < 128 {
					continue
				}
				neighbours := 0
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nx, ny := x+dx, y+dy
						if nx >= 0 && nx < w && ny >= 0 && ny < h && gray[ny*w+nx] > 127 {
							neighbours++
						}
					}
				}
				if neighbours < 2 {
					gray[y*w+x] = 0
				}
			}
		}
	}

	// Helper: decide if pixel at (x, y) should be ON
	isOn := func(x, y int) bool {
		var on bool
		if opts.Dither {
			on = gray[y*w+x] > 127
		} else {
			c := img.RGBAAt(x, y)
			if c.A < 128 {
				on = false
			} else {
				on = float64(luminance(c.R, c.G, c.B)) > float64(opts.Threshold)
			}
		}
		if opts.Invert {
			on = !on
		}
		return on
	}

	var result []byte

	switch opts.DrawMode {
	case "horizontal", "horizontal-bytes":
		// Row-major, MSB = leftmost pixel
		bytesPerRow := (w + 7) / 8
		result = make([]byte, bytesPerRow*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if isOn(x, y) {
					result[y*bytesPerRow+x/8] |= 1 << uint(7-x%8)
				}
			}
		}

	case "vertical":
		// SSD1306 page mode: row-major pages, bit0 = topmost pixel in page
		pages := (h + 7) / 8
		result = make([]byte, pages*w)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if isOn(x, y) {
					result[(y/8)*w+x] |= 1 << uint(y%8)
				}
			}
		}
	}

	return result
}

// luminance calculates perceptual grayscale value from RGB
func luminance(r, g, b uint8) uint8 {
	return uint8(float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
