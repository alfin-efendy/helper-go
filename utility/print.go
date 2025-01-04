package utility

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type LogLevel int

const (
	Info LogLevel = iota
	Trace
	Warn
	Debug
	Error
	Panic
	Fatal
)

func (l LogLevel) Color() string {
	switch l {
	case Info:
		return "#00FFFF"
	case Trace:
		return "#00FFFF"
	case Warn:
		return "#FFA500"
	case Debug:
		return "#FFA500"
	case Error:
		return "#FF0000"
	case Panic:
		return "#FF0000"
	default:
		return "#000000"
	}
}

func (l LogLevel) String() string {
	switch l {
	case Info:
		return "INFO"
	case Trace:
		return "TRACE"
	case Warn:
		return "WARNING"
	case Debug:
		return "DEBUG"
	case Error:
		return "ERROR"
	case Panic:
		return "PANIC"
	case Fatal:
		return "Fatal"
	default:
		return "UNKNOWN"
	}
}

func centerString(s string) string {
	width := 10

	if len(s) >= width {
		return s
	}
	padding := (width - len(s)) / 2
	return fmt.Sprintf("%*s%*s", padding+len(s), s, width-padding-len(s), "")
}

func standardPrint(level LogLevel, message string) {
	// Print the current date and time
	now := time.Now().Format("2006-01-02 15:04:05")
	dateFormat := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Render(now)

	// Center the log level fixed width with spaces
	levelPrint := centerString(level.String())

	levelFormat := lipgloss.NewStyle().Background(lipgloss.Color(level.Color())).Foreground(lipgloss.Color("#FFFFFF")).Render(levelPrint)

	// Message Format
	messageFormat := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render(message)

	print(dateFormat + " | " + levelFormat + " | " + messageFormat + "\n")
}

func PrintInfo(message string) {
	standardPrint(Info, message)
}

func PrintTrace(message string) {
	standardPrint(Trace, message)
}

func PrintWarning(message string) {
	standardPrint(Warn, message)
}

func PrintDebug(message string) {
	standardPrint(Debug, message)
}

func PrintError(message string) {
	standardPrint(Error, message)
}

func PrintPanic(message string) {
	standardPrint(Panic, message)
}

func PrintFatal(message string) {
	standardPrint(Fatal, message)
}
