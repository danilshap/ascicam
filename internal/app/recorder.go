package app

import (
	"fmt"
	"os"
)

type asciiRecorder struct {
	file       *os.File
	gif        *gifEncoder
	frameLimit int
	frameCount int
}

func newASCIIRecorder(opts OutputOptions, frameLimit int) (*asciiRecorder, error) {
	if opts.RecordPath == "" {
		return &asciiRecorder{}, nil
	}
	if err := ensureCaptureDir(); err != nil {
		return nil, err
	}
	encoder, err := newGIFEncoder(opts.RecordPath)
	if err != nil {
		return nil, err
	}
	return &asciiRecorder{
		gif:        encoder,
		frameLimit: frameLimit,
	}, nil
}

func (r *asciiRecorder) WriteFrame(frame string) error {
	if r.gif != nil {
		if err := r.gif.AddFrame(frame); err != nil {
			return err
		}
		r.frameCount++
		return nil
	}
	if r.file == nil {
		return nil
	}
	if r.frameCount > 0 {
		if _, err := r.file.WriteString("\f\n"); err != nil {
			return fmt.Errorf("write frame separator: %w", err)
		}
	}
	if _, err := r.file.WriteString(frame + "\n"); err != nil {
		return fmt.Errorf("write record frame: %w", err)
	}
	r.frameCount++
	return nil
}

func (r *asciiRecorder) Done() bool {
	return r.frameLimit > 0 && r.frameCount >= r.frameLimit
}

func (r *asciiRecorder) Close() error {
	if r.gif != nil {
		return r.gif.Close()
	}
	if r.file == nil {
		return nil
	}
	return r.file.Close()
}

func writeASCIIPhoto(path, frame string) error {
	if err := ensureCaptureDir(); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(frame+"\n"), 0o644); err != nil {
		return fmt.Errorf("write photo file: %w", err)
	}
	return nil
}
