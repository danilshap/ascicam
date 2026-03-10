package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gocv.io/x/gocv"
)

const frameTimeout = 3 * time.Second
const countdownStep = 700 * time.Millisecond

type RunOutcome struct {
	Notice string
}

func Run(cfg Config) (RunOutcome, error) {
	resolvedOutput, err := cfg.Output.materialize()
	if err != nil {
		return RunOutcome{}, err
	}
	cfg.Output = resolvedOutput

	webcam, err := openCamera(cfg.DeviceID)
	if err != nil {
		return RunOutcome{}, err
	}
	defer webcam.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	terminal, err := prepareTerminal(ctx, stop)
	if err != nil {
		return RunOutcome{}, err
	}
	defer terminal.Restore()

	pipeline := newFramePipeline(cfg)
	defer pipeline.Close()

	recorder, err := newASCIIRecorder(cfg.Output, cfg.maxRecordFrames())
	if err != nil {
		return RunOutcome{}, err
	}
	defer recorder.Close()

	frameDelay := time.Second / time.Duration(cfg.FPS)
	firstFrameDeadline := time.Now().Add(frameTimeout)
	manualWidth := cfg.RequestedWidth

	if needsCaptureWarmup(cfg) {
		if err := runCaptureCountdown(ctx, webcam, pipeline, terminal, frameDelay); err != nil {
			return RunOutcome{}, err
		}
	}

	for ctx.Err() == nil {
		start := time.Now()
		drainInput(terminal.Input, &manualWidth, stop)

		if ok := webcam.Read(&pipeline.frame); !ok || pipeline.frame.Empty() {
			if time.Now().After(firstFrameDeadline) {
				return RunOutcome{}, fmt.Errorf("camera opened but no frames were received within %s; check camera permissions or try --device 1", frameTimeout)
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}

		termSize := terminal.Size()
		asciiFrame, renderWidth, asciiWidth, err := pipeline.NextFrame(manualWidth, termSize)
		if err != nil {
			return RunOutcome{}, err
		}
		if asciiFrame == "" {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		paddedFrame := padFrame(asciiFrame, renderWidth)
		photoFrame, gifFrame := buildCaptureFrames(asciiFrame, termSize, cfg.Output.CaptureFullscreen)
		if err := recorder.WriteFrame(gifFrame); err != nil {
			return RunOutcome{}, err
		}
		if recorder.Done() {
			return RunOutcome{Notice: fmt.Sprintf("GIF saved: %s", cfg.Output.RecordPath)}, nil
		}
		if cfg.Output.PhotoPath != "" {
			if err := writeASCIIPhoto(cfg.Output.PhotoPath, photoFrame); err != nil {
				return RunOutcome{}, err
			}
			return RunOutcome{Notice: fmt.Sprintf("Photo saved: %s", cfg.Output.PhotoPath)}, nil
		}

		screen := strings.ReplaceAll(paddedFrame, "\n", "\r\n")
		if cfg.Render.ShowStatus {
			screen += "\r\n" + formatStatus(cfg, asciiWidth, manualWidth, renderWidth)
		}
		if _, err := fmt.Fprintf(os.Stdout, "\x1b[H%s\x1b[J", screen); err != nil {
			return RunOutcome{}, fmt.Errorf("render frame: %w", err)
		}

		if elapsed := time.Since(start); elapsed < frameDelay {
			time.Sleep(frameDelay - elapsed)
		}
	}

	return RunOutcome{}, nil
}

func needsCaptureWarmup(cfg Config) bool {
	return cfg.Output.PhotoPath != "" || cfg.Output.RecordPath != ""
}

func buildCaptureFrames(frame string, term terminalDimensions, fullscreen bool) (string, string) {
	if !fullscreen {
		return frame, frame
	}

	photoFrame := padFrameToCanvas(frame, maxInt(1, term.Width-1), maxInt(1, term.Height))
	gifFrame := padFrameToHeight(frame, maxInt(1, term.Height))
	return photoFrame, gifFrame
}

func runCaptureCountdown(ctx context.Context, webcam *gocv.VideoCapture, pipeline *framePipeline, terminal *terminalSession, frameDelay time.Duration) error {
	for _, label := range []string{"3", "2", "1"} {
		start := time.Now()
		if ok := webcam.Read(&pipeline.frame); !ok || pipeline.frame.Empty() {
			time.Sleep(40 * time.Millisecond)
		}
		if err := renderCountdownFrame(label, terminal.Size()); err != nil {
			return err
		}
		if err := sleepWithContext(ctx, countdownStep-time.Since(start)); err != nil {
			return err
		}
	}

	warmUntil := time.Now().Add(frameDelay * 2)
	for time.Now().Before(warmUntil) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_ = webcam.Read(&pipeline.frame)
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}

func renderCountdownFrame(label string, size terminalDimensions) error {
	lines := countdownArt(label)
	if len(lines) == 0 {
		return nil
	}

	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	topPadding := maxInt(0, (size.Height-len(lines))/2)
	leftPadding := maxInt(0, (size.Width-maxWidth)/2)

	var screen strings.Builder
	for i := 0; i < topPadding; i++ {
		screen.WriteString("\r\n")
	}
	for i, line := range lines {
		screen.WriteString(strings.Repeat(" ", leftPadding))
		screen.WriteString(line)
		if i < len(lines)-1 {
			screen.WriteString("\r\n")
		}
	}

	if _, err := fmt.Fprintf(os.Stdout, "\x1b[H%s\x1b[J", screen.String()); err != nil {
		return fmt.Errorf("render countdown: %w", err)
	}
	return nil
}

func countdownArt(label string) []string {
	switch label {
	case "3":
		return []string{
			" ██████╗ ",
			" ╚════██╗",
			"  █████╔╝",
			"  ╚═══██╗",
			" ██████╔╝",
			" ╚═════╝ ",
		}
	case "2":
		return []string{
			" ██████╗ ",
			" ╚════██╗",
			"  █████╔╝",
			" ██╔═══╝ ",
			" ███████╗",
			" ╚══════╝",
		}
	case "1":
		return []string{
			"   ██╗   ",
			" █████║  ",
			" ╚═██║   ",
			"   ██║   ",
			"   ██║   ",
			"   ╚═╝   ",
		}
	default:
		return nil
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func openCamera(deviceID int) (*gocv.VideoCapture, error) {
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		return nil, fmt.Errorf("open camera %d: %w", deviceID, err)
	}
	if !webcam.IsOpened() {
		webcam.Close()
		return nil, fmt.Errorf("camera %d is not available", deviceID)
	}
	webcam.Set(gocv.VideoCaptureBufferSize, 1)
	return webcam, nil
}

func formatStatus(cfg Config, asciiWidth, manualWidth, renderWidth int) string {
	widthMode := fmt.Sprintf("%d", asciiWidth)
	if manualWidth <= 0 {
		widthMode = fmt.Sprintf("auto(%d)", asciiWidth)
	}
	status := fmt.Sprintf(
		"device=%d mode=%s width=%s fps=%d contrast=%.2f brightness=%.0f invert=%t  +/- scale  0 auto  q quit",
		cfg.DeviceID,
		cfg.Render.Mode,
		widthMode,
		cfg.FPS,
		cfg.Render.Contrast,
		cfg.Render.Brightness,
		cfg.Render.Invert,
	)
	if len(status) < renderWidth {
		status += strings.Repeat(" ", renderWidth-len(status))
	}
	return status
}
