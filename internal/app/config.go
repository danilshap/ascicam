package app

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	DefaultPalette   = " .:-=+*#%@"
	DefaultFPS       = 24
	DefaultMinWidth  = 20
	StatusLineHeight = 1
	WidthStep        = 4
	MaxGIFSeconds    = 5
)

type RenderMode string

const (
	ModeGray  RenderMode = "grayscale"
	ModeEdges RenderMode = "edges"
)

type Config struct {
	UseTUI         bool
	DeviceID       int
	RequestedWidth int
	Palette        string
	FPS            int
	Mirror         bool
	Render         RenderOptions
	Output         OutputOptions
}

type RenderOptions struct {
	Mode       RenderMode
	Contrast   float64
	Brightness float64
	EdgeLow    float32
	EdgeHigh   float32
	Invert     bool
	ShowStatus bool
}

type OutputOptions struct {
	PhotoPath         string
	RecordPath        string
	CaptureFullscreen bool
}

func ParseConfig(args []string) (Config, error) {
	fs := flag.NewFlagSet("ascicam", flag.ContinueOnError)

	tui := fs.Bool("tui", false, "open the interactive terminal UI")
	deviceID := fs.Int("device", 0, "camera device index")
	width := fs.Int("width", 0, "output width in characters (0 = fit terminal)")
	palette := fs.String("palette", DefaultPalette, "ASCII palette from dark to bright")
	fps := fs.Int("fps", DefaultFPS, "maximum refresh rate")
	mirror := fs.Bool("mirror", true, "mirror the camera feed horizontally")
	mode := fs.String("mode", string(ModeGray), "render mode: grayscale or edges")
	contrast := fs.Float64("contrast", 1.15, "grayscale contrast multiplier")
	brightness := fs.Float64("brightness", 4, "grayscale brightness offset")
	edgeLow := fs.Float64("edge-low", 40, "lower Canny threshold for edge mode")
	edgeHigh := fs.Float64("edge-high", 120, "upper Canny threshold for edge mode")
	invert := fs.Bool("invert", false, "invert palette brightness mapping")
	status := fs.Bool("status", true, "show the bottom status/help line")
	photo := fs.String("photo", "", "save the first rendered ASCII frame to a .txt file and exit")
	record := fs.String("record", "", "record an animated .gif session up to 5 seconds")
	captureFullscreen := fs.Bool("capture-fullscreen", true, "save photo and gif captures padded to the full terminal size")
	fs.SetOutput(new(strings.Builder))

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{
		UseTUI:         *tui,
		DeviceID:       *deviceID,
		RequestedWidth: *width,
		Palette:        strings.TrimSpace(*palette),
		FPS:            *fps,
		Mirror:         *mirror,
		Render: RenderOptions{
			Mode:       RenderMode(strings.ToLower(strings.TrimSpace(*mode))),
			Contrast:   *contrast,
			Brightness: *brightness,
			EdgeLow:    float32(*edgeLow),
			EdgeHigh:   float32(*edgeHigh),
			Invert:     *invert,
			ShowStatus: *status,
		},
		Output: OutputOptions{
			PhotoPath:         strings.TrimSpace(*photo),
			RecordPath:        strings.TrimSpace(*record),
			CaptureFullscreen: *captureFullscreen,
		},
	}

	return cfg, cfg.NormalizeAndValidate()
}

func DefaultConfig() Config {
	cfg := Config{
		UseTUI:         true,
		DeviceID:       0,
		RequestedWidth: 0,
		Palette:        DefaultPalette,
		FPS:            DefaultFPS,
		Mirror:         true,
		Render: RenderOptions{
			Mode:       ModeGray,
			Contrast:   1.15,
			Brightness: 4,
			EdgeLow:    40,
			EdgeHigh:   120,
			Invert:     false,
			ShowStatus: true,
		},
		Output: OutputOptions{
			CaptureFullscreen: true,
		},
	}
	return cfg
}

func (c *Config) NormalizeAndValidate() error {
	if c.Palette == "" {
		c.Palette = DefaultPalette
	}
	if c.FPS <= 0 {
		c.FPS = DefaultFPS
	}
	if err := c.Render.validate(); err != nil {
		return err
	}
	if err := c.Output.validate(); err != nil {
		return err
	}
	return nil
}

func (o RenderOptions) validate() error {
	switch o.Mode {
	case ModeGray, ModeEdges:
	default:
		return fmt.Errorf("unsupported mode %q (expected grayscale or edges)", o.Mode)
	}
	if o.Contrast <= 0 {
		return errors.New("contrast must be greater than 0")
	}
	if o.EdgeLow < 0 || o.EdgeHigh < 0 {
		return errors.New("edge thresholds must be non-negative")
	}
	if o.Mode == ModeEdges && o.EdgeLow >= o.EdgeHigh {
		return errors.New("edge-low must be smaller than edge-high")
	}
	return nil
}

func (o OutputOptions) validate() error {
	if o.PhotoPath != "" && !strings.EqualFold(filepath.Ext(o.PhotoPath), ".txt") {
		return errors.New("photo output must use a .txt path")
	}
	if o.RecordPath != "" && !strings.EqualFold(filepath.Ext(o.RecordPath), ".gif") {
		return errors.New("record output must use a .gif path")
	}
	return nil
}

func (c Config) maxRecordFrames() int {
	return c.FPS * MaxGIFSeconds
}
