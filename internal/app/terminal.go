package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

type inputCommand int

const (
	cmdQuit inputCommand = iota + 1
	cmdIncreaseWidth
	cmdDecreaseWidth
	cmdAutoWidth
)

type terminalSession struct {
	restore func()
	Input   <-chan inputCommand
}

func prepareTerminal(ctx context.Context, stop context.CancelFunc) (*terminalSession, error) {
	stdinFD := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(stdinFD)
	if err != nil {
		return nil, fmt.Errorf("enable raw terminal mode: %w", err)
	}

	if _, err := io.WriteString(os.Stdout, "\x1b[?1049h\x1b[2J\x1b[H\x1b[?25l"); err != nil {
		_ = term.Restore(stdinFD, oldState)
		return nil, fmt.Errorf("prepare terminal screen: %w", err)
	}

	inputCh := make(chan inputCommand, 8)
	go readTerminalInput(ctx, stop, inputCh)

	return &terminalSession{
		restore: func() {
			_, _ = io.WriteString(os.Stdout, "\x1b[?25h\x1b[2J\x1b[H\x1b[?1049l")
			_ = term.Restore(stdinFD, oldState)
		},
		Input: inputCh,
	}, nil
}

func readTerminalInput(ctx context.Context, stop context.CancelFunc, inputCh chan<- inputCommand) {
	buf := make([]byte, 1)
	for ctx.Err() == nil {
		if _, err := os.Stdin.Read(buf); err != nil {
			return
		}
		switch buf[0] {
		case 'q', 'Q', 3:
			select {
			case inputCh <- cmdQuit:
			default:
			}
			stop()
			return
		case '+', '=':
			select {
			case inputCh <- cmdIncreaseWidth:
			default:
			}
		case '-':
			select {
			case inputCh <- cmdDecreaseWidth:
			default:
			}
		case '0':
			select {
			case inputCh <- cmdAutoWidth:
			default:
			}
		}
	}
}

func (t *terminalSession) Restore() {
	t.restore()
}

func (t *terminalSession) Size() terminalDimensions {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return terminalDimensions{Width: 80, Height: 24}
	}
	return terminalDimensions{Width: width, Height: height}
}

func drainInput(inputCh <-chan inputCommand, width *int, stop context.CancelFunc) {
	for {
		select {
		case cmd := <-inputCh:
			switch cmd {
			case cmdQuit:
				stop()
			case cmdIncreaseWidth:
				if *width <= 0 {
					*width = DefaultMinWidth + WidthStep
				} else {
					*width += WidthStep
				}
			case cmdDecreaseWidth:
				if *width <= 0 {
					*width = maxInt(DefaultMinWidth, 80-WidthStep)
				} else {
					*width = maxInt(DefaultMinWidth, *width-WidthStep)
				}
			case cmdAutoWidth:
				*width = 0
			}
		default:
			return
		}
	}
}
