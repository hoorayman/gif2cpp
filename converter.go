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

type ConvertOptions struct {
	CanvasWidth  int
	CanvasHeight int
	Threshold    uint8
	DrawMode     string
	ScaleMode    string
	Invert       bool
	FlipH        bool
	FlipV        bool
	Rotate       int
}

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

	// 使用完整 GIF 逻辑尺寸作为合成画布
	gifBounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	composite := image.NewRGBA(gifBounds)

	for i, srcPaletted := range g.Image {
		frameBounds := srcPaletted.Bounds()

		// 获取上一帧的 disposal 方式
		disposal := byte(0)
		if i > 0 && int(i-1) < len(g.Disposal) {
			disposal = g.Disposal[i-1]
		}

		// 根据 disposal 处理画布
		switch disposal {
		case gif.DisposalBackground:
			// 清除上一帧区域为透明
			if i > 0 {
				prev := g.Image[i-1].Bounds()
				for y := prev.Min.Y; y < prev.Max.Y; y++ {
					for x := prev.Min.X; x < prev.Max.X; x++ {
						composite.SetRGBA(x, y, color.RGBA{})
					}
				}
			}
		case gif.DisposalPrevious:
			// 不修改 composite，保持上上帧状态（简化处理：保持不变）
		default:
			// gif.DisposalNone 或 0：保留当前画布
		}

		// 将当前帧绘制到 composite（注意用 frameBounds 作为目标区域）
		src := image.NewRGBA(frameBounds)
		xdraw.Draw(src, frameBounds, srcPaletted, frameBounds.Min, xdraw.Src)
		xdraw.Draw(composite, frameBounds, src, image.Point{}, xdraw.Over)

		// 缩放合成帧到目标尺寸
		scaled := scaleImage(composite, scaledW, scaledH)

		// 放置到 canvas 居中
		canvas := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
		ox := (canvasW - scaledW) / 2
		oy := (canvasH - scaledH) / 2
		xdraw.Draw(canvas, image.Rect(ox, oy, ox+scaledW, oy+scaledH), scaled, image.Point{}, xdraw.Over)

		result := transformImage(canvas, opts)
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
	default:
		return canvasW, canvasH
	}
}

func scaleImage(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

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

func imageToBytes(img *image.RGBA, opts ConvertOptions) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var result []byte

	switch opts.DrawMode {
	case "horizontal", "horizontal-bytes":
		bytesPerRow := (w + 7) / 8
		result = make([]byte, bytesPerRow*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if isPixelOn(img, x, y, opts) {
					byteIndex := y*bytesPerRow + x/8
					bitIndex := uint(7 - x%8)
					result[byteIndex] |= 1 << bitIndex
				}
			}
		}

	case "vertical":
		pages := (h + 7) / 8
		result = make([]byte, pages*w)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if isPixelOn(img, x, y, opts) {
					page := y / 8
					bitInPage := uint(y % 8)
					byteIndex := page*w + x // ✅ 行优先
					result[byteIndex] |= 1 << bitInPage
				}
			}
		}
	}

	return result
}

func isPixelOn(img *image.RGBA, x, y int, opts ConvertOptions) bool {
	c := img.RGBAAt(x, y)
	if c.A < 128 {
		// 透明像素：不亮（除非 invert）
		return opts.Invert
	}
	lum := luminance(c.R, c.G, c.B)
	isWhite := lum > opts.Threshold
	if opts.Invert {
		isWhite = !isWhite
	}
	return isWhite
}

func luminance(r, g, b uint8) uint8 {
	return uint8(float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
