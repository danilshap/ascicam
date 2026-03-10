package app

import (
	"image"

	"ascicam/internal/ascii"

	"gocv.io/x/gocv"
)

type terminalDimensions struct {
	Width  int
	Height int
}

type framePipeline struct {
	cfg       Config
	renderer  ascii.Renderer
	frame     gocv.Mat
	gray      gocv.Mat
	smoothed  gocv.Mat
	processed gocv.Mat
	mask      gocv.Mat
}

func newFramePipeline(cfg Config) *framePipeline {
	return &framePipeline{
		cfg:       cfg,
		renderer:  ascii.NewRenderer(cfg.Palette, 0.42).WithInvert(cfg.Render.Invert),
		frame:     gocv.NewMat(),
		gray:      gocv.NewMat(),
		smoothed:  gocv.NewMat(),
		processed: gocv.NewMat(),
		mask:      gocv.NewMat(),
	}
}

func (p *framePipeline) Close() {
	p.frame.Close()
	p.gray.Close()
	p.smoothed.Close()
	p.processed.Close()
	p.mask.Close()
}

func (p *framePipeline) NextFrame(manualWidth int, term terminalDimensions) (string, int, int, error) {
	if p.cfg.Mirror {
		gocv.Flip(p.frame, &p.frame, 1)
	}

	gocv.CvtColor(p.frame, &p.gray, gocv.ColorBGRToGray)
	gocv.GaussianBlur(p.gray, &p.smoothed, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
	applyTone(p.smoothed, &p.processed, p.cfg.Render.Contrast, p.cfg.Render.Brightness)
	prepareMask(p.processed, &p.mask, p.cfg.Render)

	targetWidth := resolvedTargetWidth(manualWidth, term.Width)
	maxFrameHeight := term.Height
	if p.cfg.Render.ShowStatus {
		maxFrameHeight -= StatusLineHeight
	}

	asciiWidth, asciiHeight := p.renderer.Fit(p.processed.Cols(), p.processed.Rows(), targetWidth, maxFrameHeight)
	if asciiWidth < 1 || asciiHeight < 1 {
		return "", 0, 0, nil
	}

	asciiFrame, err := renderASCIIFrame(p.renderer, p.processed, p.mask, asciiWidth, asciiHeight, p.cfg.Render)
	if err != nil {
		return "", 0, 0, err
	}

	return asciiFrame, maxInt(1, term.Width-1), asciiWidth, nil
}

func resolvedTargetWidth(manualWidth, termWidth int) int {
	targetWidth := manualWidth
	if targetWidth <= 0 {
		targetWidth = maxInt(1, termWidth-1)
	}
	if targetWidth < DefaultMinWidth {
		targetWidth = DefaultMinWidth
	}
	if targetWidth >= termWidth {
		targetWidth = maxInt(DefaultMinWidth, termWidth-1)
	}
	return targetWidth
}

func applyTone(src gocv.Mat, dst *gocv.Mat, contrast, brightness float64) {
	gocv.ConvertScaleAbs(src, dst, contrast, brightness)
}

func prepareMask(processed gocv.Mat, mask *gocv.Mat, opts RenderOptions) {
	if opts.Mode == ModeEdges {
		gocv.Canny(processed, mask, opts.EdgeLow, opts.EdgeHigh)
	}
}

func renderASCIIFrame(renderer ascii.Renderer, processed, mask gocv.Mat, width, height int, opts RenderOptions) (string, error) {
	if opts.Mode == ModeEdges {
		return renderer.FrameWithMask(processed, mask, width, height)
	}
	return renderer.Frame(processed, width, height)
}
