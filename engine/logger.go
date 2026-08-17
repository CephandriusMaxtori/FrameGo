package engine

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Logger is a minimal thread-safe writer used across the engine.
type Logger struct {
	w  io.Writer
	mu sync.Mutex
}

// NewLogger returns a Logger writing to w.
func NewLogger(w io.Writer) *Logger {
	return &Logger{w: w}
}

// Printf writes a formatted info line.
func (l *Logger) Printf(format string, a ...any) {
	l.write("info", format, a...)
}

// Errorf writes a formatted error line.
func (l *Logger) Errorf(format string, a ...any) {
	l.write("error", format, a...)
}

func (l *Logger) write(level, format string, a ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w == nil {
		return
	}
	fmt.Fprintf(l.w, "%s [%s] %s\n", time.Now().Format(time.RFC3339), level, fmt.Sprintf(format, a...))
}
