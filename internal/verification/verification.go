package verification

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultTimeout = 2 * time.Minute
	maxOutputBytes = 128 * 1024
)

type Result struct {
	Command  []string
	Output   string
	Passed   bool
	ExitCode int
	TimedOut bool
}

func Run(root string, command []string) Result {
	result := Result{Command: append([]string(nil), command...), ExitCode: -1}
	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		result.Output = "No verification command configured."
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = root
	buffer := &boundedBuffer{limit: maxOutputBytes}
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	err := cmd.Run()
	result.Output = strings.TrimSpace(buffer.String())
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		if result.Output == "" {
			result.Output = fmt.Sprintf("Verification timed out after %s.", defaultTimeout)
		}
		return result
	}
	if err == nil {
		result.Passed = true
		result.ExitCode = 0
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else if result.Output == "" {
		result.Output = err.Error()
	}
	return result
}

type boundedBuffer struct {
	data      []byte
	limit     int
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if len(p) <= remaining {
			b.data = append(b.data, p...)
		} else {
			b.data = append(b.data, p[:remaining]...)
			b.truncated = true
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	text := string(b.data)
	if b.truncated {
		text += "\n... [verification output truncated] ..."
	}
	return text
}
