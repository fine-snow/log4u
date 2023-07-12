// Log function core code

package log4u

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"
)

type LogLevel uint8

const (
	ErrorLevel LogLevel = iota
	WarnLevel
	InfoLevel
	OutLevel
)

var (
	outLog   *log.Logger = nil
	infoLog  *log.Logger = nil
	warnLog  *log.Logger = nil
	errorLog *log.Logger = nil
)

func init() {
	_, err := os.Stat("./log")
	if err != nil {
		_ = os.Mkdir("log", 0777)
	}
	infoFile, err := os.OpenFile("./log/info.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	warnFile, _ := os.OpenFile("./log/warn.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	errorFile, _ := os.OpenFile("./log/error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	outLog = log.New(io.MultiWriter(infoFile, os.Stdout), "", 0)
	infoLog = log.New(io.MultiWriter(infoFile, os.Stdout), "\u001B[34mINFO\u001B[0m ", log.LstdFlags|log.Lmsgprefix|log.Llongfile)
	warnLog = log.New(io.MultiWriter(warnFile, os.Stdout), "\u001B[33mWARN\u001B[0m ", log.LstdFlags|log.Lmsgprefix|log.Llongfile)
	errorLog = log.New(io.MultiWriter(errorFile, os.Stdout), "\u001B[31mERROR\u001B[0m ", log.LstdFlags|log.Lmsgprefix)

	globalLog4u = &Log4u{o: outLog, i: infoLog, w: warnLog, e: errorLog, level: OutLevel, c: make(chan *logInfo, 100)}
	go globalLog4u.outLog()
}

type logInfo struct {
	level LogLevel
	val   *string
}

var globalLog4u *Log4u = nil

type Log4u struct {
	o     *log.Logger
	i     *log.Logger
	w     *log.Logger
	e     *log.Logger
	level LogLevel
	c     chan *logInfo
}

func New() *Log4u {
	return globalLog4u
}

func SetLevel(level LogLevel) {
	switch level {
	case OutLevel, InfoLevel, WarnLevel, ErrorLevel:
		globalLog4u.level = level
	default:
		globalLog4u.level = OutLevel
	}
}

func Wait() {
	// Sleep for 1 second to ensure that all logs are in the pipeline
	time.Sleep(time.Second)
	for {
		// All messages in the log pipeline are consumed and the infinite loop ends
		if len(globalLog4u.c) == 0 {
			break
		}
	}
}

func (l *Log4u) outLog() {
	for {
		info := <-l.c
		switch info.level {
		case OutLevel:
			_ = l.o.Output(2, *info.val)
		case InfoLevel:
			_ = l.i.Output(2, *info.val)
		case WarnLevel:
			_ = l.w.Output(2, *info.val)
		case ErrorLevel:
			_ = l.e.Output(2, *info.val)
		default:
		}
	}
}

func (l *Log4u) OUT(v ...any) {
	if l.level < OutLevel {
		return
	}
	str := fmt.Sprintln(v...)
	l.c <- &logInfo{level: OutLevel, val: &str}
}

func (l *Log4u) INFO(v ...any) {
	if l.level < InfoLevel {
		return
	}
	str := fmt.Sprintln(v...)
	l.c <- &logInfo{level: InfoLevel, val: &str}
}

func (l *Log4u) INFOF(format string, v ...any) {
	if l.level < InfoLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	l.c <- &logInfo{level: InfoLevel, val: &str}
}

func (l *Log4u) WARN(v ...any) {
	if l.level < WarnLevel {
		return
	}
	str := fmt.Sprintln(v...)
	l.c <- &logInfo{level: WarnLevel, val: &str}
}

func (l *Log4u) WARNF(format string, v ...any) {
	if l.level < WarnLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	l.c <- &logInfo{level: WarnLevel, val: &str}
}

func (l *Log4u) ERROR(v ...any) {
	str := fmt.Sprintln(v...)
	_, file, _, _ := runtime.Caller(2)
	fmt.Println(file)
	l.c <- &logInfo{level: ErrorLevel, val: &str}
}

func (l *Log4u) ERRORF(format string, v ...any) {
	str := fmt.Sprintf(format, v...)
	l.c <- &logInfo{level: ErrorLevel, val: &str}
}

func OUT(v ...any) {
	if globalLog4u.level < OutLevel {
		return
	}
	str := fmt.Sprintln(v...)
	globalLog4u.c <- &logInfo{level: OutLevel, val: &str}
}

func INFO(v ...any) {
	if globalLog4u.level < InfoLevel {
		return
	}
	str := fmt.Sprintln(v...)
	globalLog4u.c <- &logInfo{level: InfoLevel, val: &str}
}

func INFOF(format string, v ...any) {
	if globalLog4u.level < InfoLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	globalLog4u.c <- &logInfo{level: InfoLevel, val: &str}
}

func WARN(v ...any) {
	if globalLog4u.level < WarnLevel {
		return
	}
	str := fmt.Sprintln(v...)
	globalLog4u.c <- &logInfo{level: WarnLevel, val: &str}
}

func WARNF(format string, v ...any) {
	if globalLog4u.level < WarnLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	globalLog4u.c <- &logInfo{level: WarnLevel, val: &str}
}

func ERROR(v ...any) {
	str := fmt.Sprintln(v...)
	globalLog4u.c <- &logInfo{level: ErrorLevel, val: &str}
}

func ERRORF(format string, v ...any) {
	str := fmt.Sprintf(format, v...)
	globalLog4u.c <- &logInfo{level: ErrorLevel, val: &str}
}
