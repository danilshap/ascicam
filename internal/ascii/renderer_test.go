package ascii

import (
	"testing"

	"gocv.io/x/gocv"
)

func TestFitRespectsHeightLimit(t *testing.T) {
	renderer := NewRenderer(" .#@", 0.5)

	width, height := renderer.Fit(640, 480, 120, 20)

	if width != 53 {
		t.Fatalf("expected width 53, got %d", width)
	}
	if height != 19 {
		t.Fatalf("expected height 19, got %d", height)
	}
}

func TestFrameMapsBrightnessToPalette(t *testing.T) {
	renderer := NewRenderer(" .#", 1)
	mat, err := gocv.NewMatFromBytes(1, 3, gocv.MatTypeCV8UC1, []byte{0, 127, 255})
	if err != nil {
		t.Fatalf("create mat: %v", err)
	}
	defer mat.Close()

	frame, err := renderer.Frame(mat, 3, 1)
	if err != nil {
		t.Fatalf("render frame: %v", err)
	}

	if frame != "  #" {
		t.Fatalf("expected %q, got %q", "  #", frame)
	}
}

func TestFrameWithMaskBlanksMaskedPixels(t *testing.T) {
	renderer := NewRenderer(" .#", 1)
	gray, err := gocv.NewMatFromBytes(1, 3, gocv.MatTypeCV8UC1, []byte{255, 255, 255})
	if err != nil {
		t.Fatalf("create gray mat: %v", err)
	}
	defer gray.Close()

	mask, err := gocv.NewMatFromBytes(1, 3, gocv.MatTypeCV8UC1, []byte{255, 0, 255})
	if err != nil {
		t.Fatalf("create mask mat: %v", err)
	}
	defer mask.Close()

	frame, err := renderer.FrameWithMask(gray, mask, 3, 1)
	if err != nil {
		t.Fatalf("render frame: %v", err)
	}

	if frame != "# #" {
		t.Fatalf("expected %q, got %q", "# #", frame)
	}
}

func TestWithInvertReversesPaletteMapping(t *testing.T) {
	renderer := NewRenderer(" .#", 1).WithInvert(true)
	mat, err := gocv.NewMatFromBytes(1, 2, gocv.MatTypeCV8UC1, []byte{0, 255})
	if err != nil {
		t.Fatalf("create mat: %v", err)
	}
	defer mat.Close()

	frame, err := renderer.Frame(mat, 2, 1)
	if err != nil {
		t.Fatalf("render frame: %v", err)
	}

	if frame != "# " {
		t.Fatalf("expected %q, got %q", "# ", frame)
	}
}
