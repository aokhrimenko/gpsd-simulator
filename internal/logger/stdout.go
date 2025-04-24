package logger

import (
	"fmt"
)

func NewStdoutLogger(level Level) Logger {
	return &stdout{level: level}
}

type stdout struct {
	level Level
}

func (l stdout) Debugf(format string, args ...interface{}) {
	if shouldLog(LevelDebug, l.level) {
		return
	}
	fmt.Print("[DEBUG] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Debug(args ...interface{}) {
	if shouldLog(LevelDebug, l.level) {
		return
	}
	fmt.Print("[DEBUG] ")
	fmt.Println(args...)
}
func (l stdout) Infof(format string, args ...interface{}) {
	if shouldLog(LevelInfo, l.level) {
		return
	}
	fmt.Print("[INFO ] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Info(args ...interface{}) {
	if shouldLog(LevelInfo, l.level) {
		return
	}
	fmt.Print("[INFO ] ")
	fmt.Println(args...)
}
func (l stdout) Warnf(format string, args ...interface{}) {
	if shouldLog(LevelWarn, l.level) {
		return
	}
	fmt.Print("[WARN ] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Warn(args ...interface{}) {
	if shouldLog(LevelWarn, l.level) {
		return
	}
	fmt.Print("[WARN ] ")
	fmt.Println(args...)
}
func (l stdout) Errorf(format string, args ...interface{}) {
	if shouldLog(LevelError, l.level) {
		return
	}
	fmt.Print("[ERROR] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Error(args ...interface{}) {
	if shouldLog(LevelError, l.level) {
		return
	}
	fmt.Print("[ERROR] ")
	fmt.Println(args...)
}
func (l stdout) Fatalf(format string, args ...interface{}) {
	if shouldLog(LevelFatal, l.level) {
		return
	}
	fmt.Print("[FATAL] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Fatal(args ...interface{}) {
	if shouldLog(LevelFatal, l.level) {
		return
	}
	fmt.Print("[FATAL] ")
	fmt.Println(args...)
}
func (l stdout) Rawf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Raw(args ...interface{}) {
	fmt.Println(args...)
}
