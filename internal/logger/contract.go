package logger

type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Debug(args ...interface{})
	Info(args ...interface{})
	Warnf(format string, args ...interface{})
	Warn(args ...interface{})
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
}
