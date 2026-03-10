# ascicam

Minimal Go CLI that opens your webcam and renders it as live ASCII art in the terminal.
It now supports tone controls, palette inversion, and an edge-only rendering mode for a cleaner, more stylized feed.

## Project structure

```text
ascicam/
├── cmd/ascicam/main.go
├── internal/ascii/renderer.go
├── go.mod
└── README.md
```

- `cmd/ascicam/main.go`: CLI flags, webcam loop, image processing, terminal setup, quit handling.
- `internal/ascii/renderer.go`: grayscale resize, aspect-ratio fitting, brightness-to-ASCII mapping.
- `internal/ascii/renderer_test.go`: renderer behavior tests.

## Libraries

- `gocv.io/x/gocv`: webcam capture and image processing. This is the most practical MVP option in Go because it gives you camera access plus grayscale/resize operations in one package.
- `golang.org/x/term`: raw terminal mode and terminal size detection so `q` can quit immediately and the output can fit the console.
- ANSI escape sequences: used directly for clearing/redrawing the terminal without bringing in a full terminal UI framework.

## Install dependencies

This project depends on OpenCV through `gocv`.

### macOS

```bash
brew install pkg-config opencv
cd ascicam
go mod tidy
```

If `gocv` cannot find OpenCV automatically, export:

```bash
export CGO_CFLAGS="-I$(brew --prefix opencv)/include/opencv4"
export CGO_LDFLAGS="-L$(brew --prefix opencv)/lib -lopencv_core -lopencv_imgproc -lopencv_videoio"
```

### Ubuntu / Debian

```bash
sudo apt update
sudo apt install -y libopencv-dev pkg-config
cd ascicam
go mod tidy
```

### Camera permissions

- macOS: grant Terminal or your terminal app camera access in System Settings.
- Linux: ensure your user can access `/dev/video*`.

## Run

```bash
cd ascicam
go run ./cmd/ascicam
```

Useful flags:

```bash
go run ./cmd/ascicam --width 100
go run ./cmd/ascicam --device 1
go run ./cmd/ascicam --fps 15
go run ./cmd/ascicam --palette "@%#*+=-:. "
go run ./cmd/ascicam --width 72 --mirror=true
go run ./cmd/ascicam --invert
go run ./cmd/ascicam --contrast 1.35 --brightness 12
go run ./cmd/ascicam --mode edges --palette " .:+#@"
go run ./cmd/ascicam --mode edges --edge-low 30 --edge-high 90
```

## Notes

- The default camera is opened with `--device 0`. On laptops this is often the front-facing camera, but device ordering is OS-specific.
- The renderer compensates for terminal character proportions with a `0.5` height multiplier so the image looks less vertically stretched.
- The terminal is redrawn from the top-left each frame instead of fully clearing on every loop, which reduces flicker.
- `--contrast` and `--brightness` are applied before ASCII conversion, which makes dim webcams noticeably easier to read.
- `--mode edges` keeps only Canny edges, which works well for a cleaner terminal aesthetic and low-detail scenes.
- `--invert` flips the palette mapping, which is useful when pairing a dark palette with bright subjects.
- `--status=false` hides the bottom help/status line for a cleaner fullscreen look.

## Quit

- Press `q`
- Or use `Ctrl+C`
- Press `+` or `-` to adjust width while running
- Press `0` to return to automatic terminal-fit width
