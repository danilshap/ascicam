package app

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var gifPalette = color.Palette{
	color.RGBA{R: 247, G: 243, B: 232, A: 255},
	color.RGBA{R: 38, G: 34, B: 28, A: 255},
}

type gifEncoder struct {
	path   string
	images []*image.Paletted
	delays []int
}

func newGIFEncoder(path string) (*gifEncoder, error) {
	return &gifEncoder{path: path}, nil
}

func (g *gifEncoder) AddFrame(frame string) error {
	img, err := renderASCIIToPaletted(frame)
	if err != nil {
		return err
	}
	g.images = append(g.images, img)
	g.delays = append(g.delays, 4)
	return nil
}

func (g *gifEncoder) Close() error {
	if len(g.images) == 0 {
		return nil
	}
	if err := ensureCaptureDir(); err != nil {
		return err
	}
	file, err := os.Create(g.path)
	if err != nil {
		return fmt.Errorf("create gif file: %w", err)
	}
	defer file.Close()

	if err := gif.EncodeAll(file, &gif.GIF{
		Image: g.images,
		Delay: g.delays,
	}); err != nil {
		return fmt.Errorf("encode gif: %w", err)
	}
	return nil
}

func writeGIFPhoto(path, frame string) error {
	img, err := renderASCIIToPaletted(frame)
	if err != nil {
		return err
	}
	if err := ensureCaptureDir(); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create gif photo: %w", err)
	}
	defer file.Close()

	if err := gif.Encode(file, img, nil); err != nil {
		return fmt.Errorf("encode gif photo: %w", err)
	}
	return nil
}

func renderASCIIToPaletted(frame string) (*image.Paletted, error) {
	lines := normalizedGIFLines(frame)
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty ASCII frame")
	}

	face := basicfont.Face7x13
	charWidth := face.Advance
	charHeight := face.Height
	maxCols := 0
	for _, line := range lines {
		contentWidth := len(line)
		if contentWidth > maxCols {
			maxCols = contentWidth
		}
	}
	if maxCols == 0 {
		maxCols = 1
	}

	scaleX := 1
	scaleY := 2
	padding := 12
	width := maxCols*charWidth*scaleX + padding*2
	height := len(lines)*charHeight*scaleY + padding*2
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{gifPalette[0]}, image.Point{}, draw.Src)

	drawer := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(gifPalette[1]),
		Face: face,
	}

	for i, line := range lines {
		x := padding
		y := padding + face.Ascent*scaleY + i*charHeight*scaleY
		for dx := 0; dx < scaleX; dx++ {
			for dy := 0; dy < scaleY; dy++ {
				drawer.Dot = fixed.P(x+dx, y+dy)
				drawer.DrawString(line)
			}
		}
	}

	paletted := image.NewPaletted(rgba.Bounds(), gifPalette)
	draw.FloydSteinberg.Draw(paletted, rgba.Bounds(), rgba, image.Point{})
	return paletted, nil
}

func normalizedGIFLines(frame string) []string {
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}

	lastContent := -1
	for i, line := range lines {
		if line != "" {
			lastContent = i
		}
	}
	if lastContent >= 0 {
		lines = lines[:lastContent+1]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
