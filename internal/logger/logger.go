package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"golang.org/x/term"
)

const (
	// minimum fallback terminal width
	defaultWidth = 80

	// prefix layout: "  HH:MM:SS  LEVEL  "
	prefixLen = 20
)

// level colors and labels
var (
	infoLabel    = color.New(color.FgCyan, color.Bold).Sprint("INFO ")
	successLabel = color.New(color.FgGreen, color.Bold).Sprint(" OK  ")
	warnLabel    = color.New(color.FgYellow, color.Bold).Sprint("WARN ")
	errorLabel   = color.New(color.FgRed, color.Bold).Sprint("ERR  ")
	eventLabel   = color.New(color.FgMagenta, color.Bold).Sprint("EVENT")
	moveLabel    = color.New(color.FgBlue, color.Bold).Sprint("MOVE ")
	planLabel    = color.New(color.FgYellow, color.Bold).Sprint("PLAN ")

	dimStyle     = color.New(color.Faint)
	boldStyle    = color.New(color.Bold)
	successStyle = color.New(color.FgGreen)
	dimPath      = color.New(color.Faint)
	arrowStyle   = color.New(color.FgBlue, color.Bold)
	errorStyle   = color.New(color.FgRed)
)

// Logger is a colored, terminal-aware, goroutine-safe logger.
type Logger struct {
	mu  sync.Mutex
	out io.Writer
	err io.Writer
}

// New creates a new Logger writing to out (stdout) and err (stderr).
func New(out, err io.Writer) *Logger {
	return &Logger{out: out, err: err}
}

// TermWidth returns the current terminal column count, defaulting to 80.
func TermWidth() int {
	w, _, e := term.GetSize(int(os.Stdout.Fd()))
	if e != nil || w <= 0 {
		return defaultWidth
	}
	return w
}

// separator returns a full-width horizontal rule.
func separator() string {
	return dimStyle.Sprint(strings.Repeat("─", TermWidth()))
}

// ts returns the current time formatted as HH:MM:SS.
func ts() string {
	return dimStyle.Sprint(time.Now().Format("15:04:05"))
}

// truncate shortens s to maxRunes runes, appending "…" if truncated.
func truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	n := utf8.RuneCountInString(s)
	if n <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes-1]) + "…"
}

// line prints a formatted log line: "  HH:MM:SS  LABEL  msg"
func (l *Logger) line(w io.Writer, label, msg string) {
	l.mu.Lock()
	_, _ = fmt.Fprintf(w, "  %s  %s  %s\n", ts(), label, msg)
	l.mu.Unlock()
}

// Info logs an informational message.
func (l *Logger) Info(msg string) { l.line(l.out, infoLabel, msg) }

// Infof logs a formatted informational message.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Success logs a success message.
func (l *Logger) Success(msg string) {
	l.line(l.out, successLabel, successStyle.Sprint(msg))
}

// Successf logs a formatted success message.
func (l *Logger) Successf(format string, args ...any) {
	l.Success(fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	l.line(l.out, warnLabel, color.New(color.FgYellow).Sprint(msg))
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Error logs an error message to stderr.
func (l *Logger) Error(msg string) {
	l.line(l.err, errorLabel, errorStyle.Sprint(msg))
}

// Errorf logs a formatted error message to stderr.
func (l *Logger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

// Event logs a filesystem/watch event.
func (l *Logger) Event(msg string) {
	l.line(l.out, eventLabel, color.New(color.FgMagenta).Sprint(msg))
}

// Eventf logs a formatted event message.
func (l *Logger) Eventf(format string, args ...any) {
	l.Event(fmt.Sprintf(format, args...))
}

// Move logs a file move operation with src → destination, truncating paths to fit.
func (l *Logger) Move(src, destination string) {
	// "  HH:MM:SS  MOVE   <src> → <destination>"
	// fixed visible chars: 2 + 8 + 2 + 5 + 2 = 19, plus " → " = 3 → 22
	available := TermWidth() - prefixLen - 3 // 3 for " → "
	half := available / 2

	srcTrim := truncate(src, half)
	destinationTrim := truncate(destination, half)

	arrow := arrowStyle.Sprint("→")
	msg := fmt.Sprintf("%s %s %s", dimPath.Sprint(srcTrim), arrow, successStyle.Sprint(destinationTrim))
	l.line(l.out, moveLabel, msg)
}

// Plan logs a move that would happen in dry-run mode: src → destination, using
// the same layout as Move but a distinct yellow "PLAN" label so previews are
// never mistaken for real moves.
func (l *Logger) Plan(src, destination string) {
	available := TermWidth() - prefixLen - 3 // 3 for " → "
	half := available / 2

	srcTrim := truncate(src, half)
	destinationTrim := truncate(destination, half)

	arrow := arrowStyle.Sprint("→")
	msg := fmt.Sprintf("%s %s %s", dimPath.Sprint(srcTrim), arrow, color.New(color.FgYellow).Sprint(destinationTrim))
	l.line(l.out, planLabel, msg)
}

// Separator prints a full-width horizontal rule to stdout.
func (l *Logger) Separator() {
	s := separator()
	l.mu.Lock()
	_, _ = fmt.Fprintln(l.out, s)
	l.mu.Unlock()
}

// Header prints a styled banner with the app name and optional version.
func (l *Logger) Header(appName, version string) {
	width := TermWidth()
	sep := dimStyle.Sprint(strings.Repeat("─", width))

	title := boldStyle.Sprint(appName)
	ver := ""
	if version != "" {
		ver = "  " + dimStyle.Sprint("v"+version)
	}
	padding := strings.Repeat(" ", 2)

	l.mu.Lock()
	_, _ = fmt.Fprintf(l.out, "\n%s\n", sep)
	_, _ = fmt.Fprintf(l.out, "%s%s%s\n", padding, title, ver)
	_, _ = fmt.Fprintf(l.out, "%s\n\n", sep)
	l.mu.Unlock()
}

// Footer prints a closing separator to stdout.
func (l *Logger) Footer() {
	s := separator()
	l.mu.Lock()
	_, _ = fmt.Fprintf(l.out, "\n%s\n\n", s)
	l.mu.Unlock()
}

// Println implements a log.Logger-compatible method (maps to Info).
func (l *Logger) Println(args ...any) {
	l.Info(fmt.Sprint(args...))
}

// Printf implements a log.Logger-compatible method (maps to Info).
func (l *Logger) Printf(format string, args ...any) {
	l.mu.Lock()
	_, _ = fmt.Fprintf(l.out, format, args...)
	l.mu.Unlock()
}
