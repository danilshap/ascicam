# `ascicam`

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)
![OpenCV](https://img.shields.io/badge/OpenCV-gocv-5C3EE8?style=flat-square&logo=opencv)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-222222?style=flat-square)
![Interface](https://img.shields.io/badge/interface-terminal%20ASCII-F7DF1E?style=flat-square&logo=gnu-bash&logoColor=111111)

```text
   __ _ ___  ___ _  ___ __ _ _ __ ___
  / _` / __|/ __| |/ __/ _` | '_ ` _ \
 | (_| \__ \ (__| | (_| (_| | | | | | |
  \__,_|___/\___|_|\___\__,_|_| |_| |_|

          live webcam -> grayscale -> terminal ASCII
             Go + OpenCV + ANSI terminal rendering
```

Turn your webcam into a live ASCII feed in the terminal.

`ascicam` is a small Go CLI for people who want a camera preview that feels a bit more interesting than a plain window: fast startup, no GUI framework, adjustable tone, edge mode, and a terminal-first workflow.

Running it without flags opens a pane-based terminal UI, so normal use no longer depends on remembering command-line options.

## ✨ Features

- Live webcam rendering directly in the terminal
- ASCII palette mapping with custom character ramps
- `grayscale` and `edges` render modes
- Contrast, brightness, and palette inversion controls
- Auto-fit sizing plus runtime width adjustment
- ASCII photo export to `.txt`
- GIF session recording up to 5 seconds
- Pane-based terminal UI with keyboard navigation
- Capture countdown and saved-file confirmation
- Raw-terminal controls with low-flicker redraw

## 🎞️ Demo Feel

```text
..::--==++**##%%@@
..::--==+++**##%%@
  ...:::---==++**#
```

Works well for:

- terminal demos
- SSH-friendly visual debugging
- weird little camera tools
- retro/CLI aesthetics

## 🛠️ Install

`ascicam` uses [`gocv`](https://gocv.io/) and therefore needs OpenCV installed locally.

### 🍎 macOS

```bash
brew install pkg-config opencv
go mod tidy
```

If `gocv` does not detect OpenCV automatically:

```bash
export CGO_CFLAGS="-I$(brew --prefix opencv)/include/opencv4"
export CGO_LDFLAGS="-L$(brew --prefix opencv)/lib -lopencv_core -lopencv_imgproc -lopencv_videoio"
```

### 🐧 Ubuntu / Debian

```bash
sudo apt update
sudo apt install -y libopencv-dev pkg-config
go mod tidy
```

### 📷 Camera Access

- On macOS, grant camera permission to your terminal app.
- On Linux, make sure your user can access `/dev/video*`.

## ▶️ Run

```bash
go run ./cmd/ascicam
```

### Quick Recipes

Open the terminal UI:

```bash
go run ./cmd/ascicam
```

Force the terminal UI explicitly:

```bash
go run ./cmd/ascicam --tui
```

Direct preview with flags:

```bash
go run ./cmd/ascicam --mode grayscale --width 100
```

Sharper grayscale:

```bash
go run ./cmd/ascicam --contrast 1.35 --brightness 12
```

Edge-only terminal look:

```bash
go run ./cmd/ascicam --mode edges --palette " .:+#@" --edge-low 30 --edge-high 90
```

Wide mirrored feed:

```bash
go run ./cmd/ascicam --width 100 --mirror=true
```

Inverted palette:

```bash
go run ./cmd/ascicam --invert
```

Save an ASCII photo:

```bash
go run ./cmd/ascicam --photo frame.txt
```

Record an animated GIF:

```bash
go run ./cmd/ascicam --record session.gif
```

## ⚙️ Flags

```text
--tui          open the terminal UI
--device       camera device index
--width        output width in characters, 0 = auto-fit
--fps          maximum refresh rate
--palette      ASCII ramp from dark to bright
--mirror       mirror the camera horizontally
--mode         grayscale | edges
--contrast     grayscale contrast multiplier
--brightness   grayscale brightness offset
--edge-low     lower Canny threshold for edge mode
--edge-high    upper Canny threshold for edge mode
--invert       invert palette brightness mapping
--status       show or hide the bottom status line
--photo        save the first rendered frame to a .txt file and exit
--record       record an animated .gif session up to 5 seconds
--capture-fullscreen save photo and gif captures padded to full terminal size
```

## ⌨️ Runtime Controls

While the app is running:

- `q` quits
- `Ctrl+C` quits
- `+` and `-` change width
- `0` returns to auto width

## 📝 Notes

- Device ordering depends on the OS and hardware. `--device 0` is just the default guess.
- Character cells are taller than they are wide, so the renderer compensates for aspect ratio.
- `edges` mode uses Canny edge detection and works best when you want bold outlines instead of smooth shading.
- `--status=false` gives a cleaner fullscreen terminal look.
- `--photo` writes plain text only.
- `--record` writes an animated GIF only and stops automatically after 5 seconds.
- `--capture-fullscreen=true` is the default, so saved captures fill the current terminal canvas.
- Saved captures are written into `captures/` with a timestamp prefix in the filename.
- After saving a photo or GIF, the app shows a confirmation with the saved path.
- Running without flags opens the pane-based terminal UI.

## 📦 Project Layout

```text
ascicam/
├── cmd/ascicam/main.go
├── internal/app/
├── internal/ascii/renderer.go
├── internal/ascii/renderer_test.go
├── go.mod
└── README.md
```

- `cmd/ascicam/main.go`: thin entry point.
- `internal/app/`: application layer for config, Bubble Tea TUI, terminal session, frame pipeline, and recording.
- `internal/ascii/renderer.go`: ASCII mapping, resizing, and masked rendering.
- `internal/ascii/renderer_test.go`: renderer behavior tests.
