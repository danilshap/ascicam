package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CaptureDir = "captures"

func (o OutputOptions) materialize() (OutputOptions, error) {
	resolved := o

	if o.PhotoPath != "" {
		path, err := buildCapturePath(o.PhotoPath)
		if err != nil {
			return OutputOptions{}, err
		}
		resolved.PhotoPath = path
	}
	if o.RecordPath != "" {
		path, err := buildCapturePath(o.RecordPath)
		if err != nil {
			return OutputOptions{}, err
		}
		resolved.RecordPath = path
	}

	return resolved, nil
}

func buildCapturePath(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", nil
	}

	base := filepath.Base(name)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "", fmt.Errorf("invalid output name %q", raw)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s", timestamp, base)
	return filepath.Join(CaptureDir, filename), nil
}

func ensureCaptureDir() error {
	if err := os.MkdirAll(CaptureDir, 0o755); err != nil {
		return fmt.Errorf("create capture directory: %w", err)
	}
	return nil
}
