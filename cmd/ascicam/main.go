package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ascicam/internal/ascii"

	"gocv.io/x/gocv"
	"golang.org/x/term"
)

const (
	defaultPalette   = " .:-=+*#%@"
	defaultFPS       = 24
	defaultMinWidth  = 20
	statusLineHeight = 1
	frameTimeout     = 3 * time.Second
	widthStep        = 4
)

type renderMode string

const (
	modeGray  renderMode = "grayscale"
	modeEdges renderMode = "edges"
)

type renderOptions struct {
	mode       renderMode
	contrast   float64
	brightness float64
	edgeLow    float32
	edgeHigh   float32
	invert     bool
	showStatus bool
}

type inputCommand int

const (
	cmdQuit inputCommand = iota + 1
	cmdIncreaseWidth
	cmdDecreaseWidth
	cmdAutoWidth
)

func main() {
	deviceID := flag.Int("device", 0, "camera device index")
	width := flag.Int("width", 0, "output width in characters (0 = fit terminal)")
	palette := flag.String("palette", defaultPalette, "ASCII palette from dark to bright")
	fps := flag.Int("fps", defaultFPS, "maximum refresh rate")
	mirror := flag.Bool("mirror", true, "mirror the camera feed horizontally")
	mode := flag.String("mode", string(modeGray), "render mode: grayscale or edges")
	contrast := flag.Float64("contrast", 1.15, "grayscale contrast multiplier")
	brightness := flag.Float64("brightness", 4, "grayscale brightness offset")
	edgeLow := flag.Float64("edge-low", 40, "lower Canny threshold for edge mode")
	edgeHigh := flag.Float64("edge-high", 120, "upper Canny threshold for edge mode")
	invert := flag.Bool("invert", false, "invert palette brightness mapping")
	status := flag.Bool("status", true, "show the bottom status/help line")
	flag.Parse()

	opts := renderOptions{
		mode:       renderMode(strings.ToLower(strings.TrimSpace(*mode))),
		contrast:   *contrast,
		brightness: *brightness,
		edgeLow:    float32(*edgeLow),
		edgeHigh:   float32(*edgeHigh),
		invert:     *invert,
		showStatus: *status,
	}

	if err := run(*deviceID, *width, *palette, *fps, *mirror, opts); err != nil {
		fmt.Fprintf(os.Stderr, "ascicam: %v\n", err)
		os.Exit(1)
	}
}

func run(deviceID, requestedWidth int, palette string, fps int, mirror bool, opts renderOptions) error {
	if strings.TrimSpace(palette) == "" {
		palette = defaultPalette
	}
	if fps <= 0 {
		fps = defaultFPS
	}
	if err := opts.validate(); err != nil {
		return err
	}

	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		return fmt.Errorf("open camera %d: %w", deviceID, err)
	}
	defer webcam.Close()

	if !webcam.IsOpened() {
		return fmt.Errorf("camera %d is not available", deviceID)
	}

	webcam.Set(gocv.VideoCaptureBufferSize, 1)

	renderer := ascii.NewRenderer(palette, 0.42).WithInvert(opts.invert)
	frame := gocv.NewMat()
	gray := gocv.NewMat()
	smoothed := gocv.NewMat()
	processed := gocv.NewMat()
	mask := gocv.NewMat()
	defer frame.Close()
	defer gray.Close()
	defer smoothed.Close()
	defer processed.Close()
	defer mask.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	restoreTerminal, inputCh, keyErr := prepareTerminal(ctx, stop)
	if keyErr != nil {
		return keyErr
	}
	defer restoreTerminal()

	frameDelay := time.Second / time.Duration(fps)
	firstFrameDeadline := time.Now().Add(frameTimeout)
	manualWidth := requestedWidth

	for ctx.Err() == nil {
		start := time.Now()
		drainInput(inputCh, &manualWidth, stop)

		if ok := webcam.Read(&frame); !ok || frame.Empty() {
			if time.Now().After(firstFrameDeadline) {
				return fmt.Errorf("camera opened but no frames were received within %s; check camera permissions or try --device 1", frameTimeout)
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}

		if mirror {
			gocv.Flip(frame, &frame, 1)
		}

		gocv.CvtColor(frame, &gray, gocv.ColorBGRToGray)
		gocv.GaussianBlur(gray, &smoothed, image.Pt(5, 5), 0, 0, gocv.BorderDefault)
		applyTone(smoothed, &processed, opts.contrast, opts.brightness)
		prepareMask(processed, &mask, opts)

		termWidth, termHeight := terminalSize()
		targetWidth := manualWidth
		if targetWidth <= 0 {
			targetWidth = max(1, termWidth-1)
		}
		if targetWidth < defaultMinWidth {
			targetWidth = defaultMinWidth
		}
		if targetWidth >= termWidth {
			targetWidth = max(defaultMinWidth, termWidth-1)
		}

		maxFrameHeight := termHeight
		if opts.showStatus {
			maxFrameHeight -= statusLineHeight
		}
		asciiWidth, asciiHeight := renderer.Fit(processed.Cols(), processed.Rows(), targetWidth, maxFrameHeight)
		if asciiWidth < 1 || asciiHeight < 1 {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		asciiFrame, err := renderASCIIFrame(renderer, processed, mask, asciiWidth, asciiHeight, opts)
		if err != nil {
			return err
		}
		renderWidth := max(1, termWidth-1)
		asciiFrame = padFrame(asciiFrame, renderWidth)
		asciiFrame = strings.ReplaceAll(asciiFrame, "\n", "\r\n")

		screen := asciiFrame
		if opts.showStatus {
			screen += "\r\n" + formatStatus(deviceID, asciiWidth, fps, manualWidth, opts, renderWidth)
		}
		if _, err := fmt.Fprintf(os.Stdout, "\x1b[H%s\x1b[J", screen); err != nil {
			return fmt.Errorf("render frame: %w", err)
		}

		elapsed := time.Since(start)
		if elapsed < frameDelay {
			time.Sleep(frameDelay - elapsed)
		}
	}

	return nil
}

func (o renderOptions) validate() error {
	switch o.mode {
	case modeGray, modeEdges:
	default:
		return fmt.Errorf("unsupported mode %q (expected grayscale or edges)", o.mode)
	}
	if o.contrast <= 0 {
		return errors.New("contrast must be greater than 0")
	}
	if o.edgeLow < 0 || o.edgeHigh < 0 {
		return errors.New("edge thresholds must be non-negative")
	}
	if o.mode == modeEdges && o.edgeLow >= o.edgeHigh {
		return errors.New("edge-low must be smaller than edge-high")
	}
	return nil
}

func prepareTerminal(ctx context.Context, stop context.CancelFunc) (func(), <-chan inputCommand, error) {
	stdinFD := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(stdinFD)
	if err != nil {
		return nil, nil, fmt.Errorf("enable raw terminal mode: %w", err)
	}

	if _, err := io.WriteString(os.Stdout, "\x1b[?1049h\x1b[2J\x1b[H\x1b[?25l"); err != nil {
		_ = term.Restore(stdinFD, oldState)
		return nil, nil, fmt.Errorf("prepare terminal screen: %w", err)
	}

	inputCh := make(chan inputCommand, 8)

	go func() {
		buf := make([]byte, 1)
		for ctx.Err() == nil {
			if _, err := os.Stdin.Read(buf); err != nil {
				return
			}
			switch buf[0] {
			case 'q', 'Q', 3:
				select {
				case inputCh <- cmdQuit:
				default:
				}
				stop()
				return
			case '+', '=':
				select {
				case inputCh <- cmdIncreaseWidth:
				default:
				}
			case '-':
				select {
				case inputCh <- cmdDecreaseWidth:
				default:
				}
			case '0':
				select {
				case inputCh <- cmdAutoWidth:
				default:
				}
			}
		}
	}()

	restore := func() {
		_, _ = io.WriteString(os.Stdout, "\x1b[?25h\x1b[2J\x1b[H\x1b[?1049l")
		_ = term.Restore(stdinFD, oldState)
	}

	return restore, inputCh, nil
}

func terminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return width, height
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func drainInput(inputCh <-chan inputCommand, width *int, stop context.CancelFunc) {
	for {
		select {
		case cmd := <-inputCh:
			switch cmd {
			case cmdQuit:
				stop()
			case cmdIncreaseWidth:
				if *width <= 0 {
					*width = defaultMinWidth + widthStep
				} else {
					*width += widthStep
				}
			case cmdDecreaseWidth:
				if *width <= 0 {
					*width = max(defaultMinWidth, 80-widthStep)
				} else {
					*width = max(defaultMinWidth, *width-widthStep)
				}
			case cmdAutoWidth:
				*width = 0
			}
		default:
			return
		}
	}
}

func padFrame(frame string, width int) string {
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		if len(line) < width {
			lines[i] = line + strings.Repeat(" ", width-len(line))
		}
	}
	return strings.Join(lines, "\n")
}

func applyTone(src gocv.Mat, dst *gocv.Mat, contrast, brightness float64) {
	gocv.ConvertScaleAbs(src, dst, contrast, brightness)
}

func prepareMask(processed gocv.Mat, mask *gocv.Mat, opts renderOptions) {
	if opts.mode == modeEdges {
		gocv.Canny(processed, mask, opts.edgeLow, opts.edgeHigh)
	}
}

func renderASCIIFrame(renderer ascii.Renderer, processed, mask gocv.Mat, width, height int, opts renderOptions) (string, error) {
	if opts.mode == modeEdges {
		return renderer.FrameWithMask(processed, mask, width, height)
	}
	return renderer.Frame(processed, width, height)
}

func formatStatus(deviceID, asciiWidth, fps, manualWidth int, opts renderOptions, renderWidth int) string {
	widthMode := fmt.Sprintf("%d", asciiWidth)
	if manualWidth <= 0 {
		widthMode = fmt.Sprintf("auto(%d)", asciiWidth)
	}
	status := fmt.Sprintf(
		"device=%d mode=%s width=%s fps=%d contrast=%.2f brightness=%.0f invert=%t  +/- scale  0 auto  q quit",
		deviceID,
		opts.mode,
		widthMode,
		fps,
		opts.contrast,
		opts.brightness,
		opts.invert,
	)
	if len(status) < renderWidth {
		status += strings.Repeat(" ", renderWidth-len(status))
	}
	return status
}
