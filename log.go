package procmoni

import (
	"fmt"
	"path"
	"runtime"
	"strconv"
	"time"
)

const (
	logLevelDebug = iota
	logLevelInfo
	logLevelError
)

type Log interface {
	Debug(args ...interface{})
	Debugf(f string, args ...interface{})
	Info(args ...interface{})
	Infof(f string, args ...interface{})
	Error(args ...interface{})
	Errorf(f string, args ...interface{})
}

type stdLog struct {
	skip  int
	level int
}

func NewStdLog(level int, embedded bool) *stdLog {
	skip := 3
	if embedded {
		skip = 4
	}
	return &stdLog{
		skip:  skip,
		level: level,
	}
}

func (s stdLog) Debug(args ...interface{}) {
	s.log(logLevelDebug, fmt.Sprint(args...))
}

func (s stdLog) Debugf(f string, args ...interface{}) {
	s.log(logLevelDebug, fmt.Sprintf(f, args...))
}

func (s stdLog) Info(args ...interface{}) {
	s.log(logLevelInfo, fmt.Sprint(args...))
}

func (s stdLog) Infof(f string, args ...interface{}) {
	s.log(logLevelInfo, fmt.Sprintf(f, args...))
}

func (s stdLog) Error(args ...interface{}) {
	s.log(logLevelError, fmt.Sprint(args...))
}

func (s stdLog) Errorf(f string, args ...interface{}) {
	s.log(logLevelError, fmt.Sprintf(f, args...))
}

func (s stdLog) log(level int, text string) {
	if level < s.level {
		return
	}
	fmt.Print("\033[1;37;1m", time.Now().Format("2006-01-02T15:04:05.000"), " ")
	color := 33
	levelText := "D"
	switch level {
	case logLevelError:
		color = 31
		levelText = "E"
	case logLevelInfo:
		color = 37
		levelText = "I"
	}
	fmt.Printf("\033[1;%d;1m[%s]%s %s\033[0m\n", color, levelText, s.caller(), text)
}

func (s stdLog) caller() string {
	_, file, line, ok := runtime.Caller(s.skip)
	if !ok {
		return ""
	}
	return path.Base(file) + ":" + strconv.Itoa(line)
}
