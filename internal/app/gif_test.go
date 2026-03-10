package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderASCIIToPalettedContainsVisiblePixels(t *testing.T) {
	img, err := renderASCIIToPaletted("###\n# #\n###")
	if err != nil {
		t.Fatalf("renderASCIIToPaletted: %v", err)
	}

	seenForeground := false
	for _, idx := range img.Pix {
		if idx != 0 {
			seenForeground = true
			break
		}
	}

	if !seenForeground {
		t.Fatalf("expected rendered GIF frame to contain foreground pixels")
	}
}

func TestWriteGIFPhotoCreatesNonEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.gif")

	if err := writeGIFPhoto(path, "@@@\n@@@"); err != nil {
		t.Fatalf("writeGIFPhoto: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat gif: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty GIF file")
	}
}

func TestNormalizedGIFLinesTrimTrailingSpacePadding(t *testing.T) {
	lines := normalizedGIFLines("###     \n# #     \n        \n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after trimming, got %d", len(lines))
	}
	if lines[0] != "###" {
		t.Fatalf("expected first line to be trimmed to %q, got %q", "###", lines[0])
	}
	if lines[1] != "# #" {
		t.Fatalf("expected second line to be trimmed to %q, got %q", "# #", lines[1])
	}
}

func TestASCIIRecorderDoneWithGIFRecording(t *testing.T) {
	recorder := &asciiRecorder{gif: &gifEncoder{}, frameLimit: 2}

	if err := recorder.WriteFrame("###"); err != nil {
		t.Fatalf("first frame: %v", err)
	}
	if recorder.Done() {
		t.Fatalf("recorder should not be done after first frame")
	}
	if err := recorder.WriteFrame("###"); err != nil {
		t.Fatalf("second frame: %v", err)
	}
	if !recorder.Done() {
		t.Fatalf("recorder should be done after reaching frame limit")
	}
}
