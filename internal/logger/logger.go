package logger

type Level int

const (
	LevelFatal Level = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelVerbose
)

type Logger interface {
	Verbosef(format string, args ...interface{})
	Verbose(args ...interface{})
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})
	Infof(format string, args ...interface{})
	Info(args ...interface{})
	Warnf(format string, args ...interface{})
	Warn(args ...interface{})
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
	Rawf(format string, args ...interface{})
	Raw(args ...interface{})
}

func shouldLog(level Level, maxLevel Level) bool {
	return level > maxLevel
}
