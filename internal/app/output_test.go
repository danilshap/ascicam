package app

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputOptionsMaterializeUsesCaptureDirAndKeepsBasename(t *testing.T) {
	out, err := (OutputOptions{
		PhotoPath:  "frame.txt",
		RecordPath: "session.gif",
	}).materialize()
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	if filepath.Dir(out.PhotoPath) != CaptureDir {
		t.Fatalf("expected photo path in %q, got %q", CaptureDir, out.PhotoPath)
	}
	if filepath.Dir(out.RecordPath) != CaptureDir {
		t.Fatalf("expected record path in %q, got %q", CaptureDir, out.RecordPath)
	}
	if !strings.HasSuffix(out.PhotoPath, "_frame.txt") {
		t.Fatalf("expected photo basename to be preserved, got %q", out.PhotoPath)
	}
	if !strings.HasSuffix(out.RecordPath, "_session.gif") {
		t.Fatalf("expected record basename to be preserved, got %q", out.RecordPath)
	}
}
