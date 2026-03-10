package app

import "strings"

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func padFrameToCanvas(frame string, width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		if len(line) < width {
			lines[i] = line + strings.Repeat(" ", width-len(line))
		} else if len(line) > width {
			lines[i] = line[:width]
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func padFrameToHeight(frame string, height int) string {
	if height < 1 {
		height = 1
	}

	lines := strings.Split(frame, "\n")
	maxWidth := 1
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	for i, line := range lines {
		if len(line) < maxWidth {
			lines[i] = line + strings.Repeat(" ", maxWidth-len(line))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", maxWidth))
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
