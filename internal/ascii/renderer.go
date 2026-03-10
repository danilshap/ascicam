package ascii

import (
	"fmt"
	"image"
	"strings"

	"gocv.io/x/gocv"
)

type Renderer struct {
	palette    []byte
	charAspect float64
	invert     bool
}

func NewRenderer(palette string, charAspect float64) Renderer {
	if palette == "" {
		palette = "@#%*+=-:. "
	}
	if charAspect <= 0 {
		charAspect = 0.5
	}

	return Renderer{
		palette:    []byte(palette),
		charAspect: charAspect,
	}
}

func (r Renderer) WithInvert(invert bool) Renderer {
	r.invert = invert
	return r
}

func (r Renderer) Fit(sourceWidth, sourceHeight, requestedWidth, maxHeight int) (int, int) {
	if sourceWidth <= 0 || sourceHeight <= 0 || requestedWidth <= 0 || maxHeight <= 0 {
		return 0, 0
	}

	width := requestedWidth
	height := r.scaledHeight(sourceWidth, sourceHeight, width)
	if height <= maxHeight {
		return width, height
	}

	width = int(float64(maxHeight) * float64(sourceWidth) / (float64(sourceHeight) * r.charAspect))
	if width < 1 {
		width = 1
	}

	return width, r.scaledHeight(sourceWidth, sourceHeight, width)
}

func (r Renderer) Frame(gray gocv.Mat, width, height int) (string, error) {
	emptyMask := gocv.NewMat()
	defer emptyMask.Close()
	return r.FrameWithMask(gray, emptyMask, width, height)
}

func (r Renderer) FrameWithMask(gray gocv.Mat, mask gocv.Mat, width, height int) (string, error) {
	if gray.Empty() {
		return "", fmt.Errorf("empty grayscale frame")
	}
	if gray.Channels() != 1 {
		return "", fmt.Errorf("renderer expects a single-channel grayscale frame")
	}

	resizedGray := gocv.NewMat()
	resizedMask := gocv.NewMat()
	defer resizedGray.Close()
	defer resizedMask.Close()

	gocv.Resize(gray, &resizedGray, image.Pt(width, height), 0, 0, gocv.InterpolationArea)
	if !mask.Empty() {
		gocv.Resize(mask, &resizedMask, image.Pt(width, height), 0, 0, gocv.InterpolationArea)
	}

	var out strings.Builder
	out.Grow(height * (width + 1))

	last := len(r.palette) - 1
	for y := 0; y < resizedGray.Rows(); y++ {
		for x := 0; x < resizedGray.Cols(); x++ {
			if !resizedMask.Empty() {
				maskValue := resizedMask.GetUCharAt(y, x)
				if maskValue < 32 {
					out.WriteByte(' ')
					continue
				}
			}

			value := resizedGray.GetUCharAt(y, x)
			idx := int(value) * last / 255
			if r.invert {
				idx = last - idx
			}
			out.WriteByte(r.palette[idx])
		}
		if y < resizedGray.Rows()-1 {
			out.WriteByte('\n')
		}
	}

	return out.String(), nil
}

func (r Renderer) scaledHeight(sourceWidth, sourceHeight, targetWidth int) int {
	height := int((float64(sourceHeight) / float64(sourceWidth)) * float64(targetWidth) * r.charAspect)
	if height < 1 {
		return 1
	}
	return height
}
