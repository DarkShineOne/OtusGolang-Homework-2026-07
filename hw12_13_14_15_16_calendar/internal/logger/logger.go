package logger

import (
	"fmt"
	"io"
	"log"
	"strings"
)

type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

type Logger struct {
	level  Level
	logger *log.Logger
}

func New(level string, output io.Writer) (*Logger, error) {
	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}
	return &Logger{
		level:  parsedLevel,
		logger: log.New(output, "", log.LstdFlags|log.Lmicroseconds),
	}, nil
}

func (l *Logger) Debug(msg string) {
	l.log(LevelDebug, "DEBUG", msg)
}

func (l *Logger) Info(msg string) {
	l.log(LevelInfo, "INFO", msg)
}

func (l *Logger) Warn(msg string) {
	l.log(LevelWarn, "WARN", msg)
}

func (l *Logger) Error(msg string) {
	l.log(LevelError, "ERROR", msg)
}

func (l *Logger) log(level Level, label, msg string) {
	if level > l.level {
		return
	}
	l.logger.Printf("%-5s %s", label, msg)
}

func parseLevel(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "error":
		return LevelError, nil
	case "warn":
		return LevelWarn, nil
	case "info":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	default:
		return LevelInfo, fmt.Errorf("unsupported log level %q", level)
	}
}
