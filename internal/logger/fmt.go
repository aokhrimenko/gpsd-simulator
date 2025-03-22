package logger

import (
	"fmt"
)

func NewStdoutLogger() Logger {
	return &stdout{}
}

type stdout struct {
}

func (l stdout) Debugf(format string, args ...interface{}) {
	fmt.Print("[DEBUG] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Debug(args ...interface{}) {
	fmt.Print("[DEBUG] ")
	fmt.Println(args...)
}
func (l stdout) Infof(format string, args ...interface{}) {
	fmt.Print("[INFO ] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Info(args ...interface{}) {
	fmt.Print("[INFO ] ")
	fmt.Println(args...)
}
func (l stdout) Warnf(format string, args ...interface{}) {
	fmt.Print("[WARN ] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Warn(args ...interface{}) {
	fmt.Print("[WARN ] ")
	fmt.Println(args...)
}
func (l stdout) Errorf(format string, args ...interface{}) {
	fmt.Print("[ERROR] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Error(args ...interface{}) {
	fmt.Print("[ERROR] ")
	fmt.Println(args...)
}
func (l stdout) Fatalf(format string, args ...interface{}) {
	fmt.Print("[FATAL] ")
	fmt.Printf(format, args...)
	fmt.Println()
}
func (l stdout) Fatal(args ...interface{}) {
	fmt.Print("[FATAL] ")
	fmt.Println(args...)
}
