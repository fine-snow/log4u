// Log function core code

package log4u

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
)

// LogLevel Log level parameter type
type LogLevel uint8

const (
	ErrorLevel LogLevel = iota
	WarnLevel
	InfoLevel
	OutLevel
)

const (
	Ldate         = 1 << iota     // the date in the local time zone: 2009-01-23
	Ltime                         // the time in the local time zone: 01:23:23
	Lmicroseconds                 // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                     // full file name and line number: /a/b/c/d.go:23
	Lshortfile                    // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                          // if Ldate or Ltime is set, use UTC rather than the local time zone
	Lmsgprefix                    // move the "prefix" from the beginning of the line to before the message
	LstdFlags     = Ldate | Ltime // initial values for the standard logger
)

type Logger struct {
	prefix string
	flag   int
	out    io.Writer
	buf    []byte
}

func newLogger(out io.Writer, prefix string, flag int) *Logger {
	l := &Logger{out: out, prefix: prefix, flag: flag}
	return l
}

func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (l *Logger) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	if l.flag&Lmsgprefix == 0 {
		*buf = append(*buf, l.prefix...)
	}
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&LUTC != 0 {
			t = t.UTC()
		}
		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '-')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '-')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if l.flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if l.flag&(Lshortfile|Llongfile) != 0 {
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ": "...)
	}
	if l.flag&Lmsgprefix != 0 {
		*buf = append(*buf, l.prefix...)
	}
}

func (l *Logger) Output(s, file string, line int) {
	// get this early
	now := time.Now()
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, file, line)
	l.buf = append(l.buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		l.buf = append(l.buf, '\n')
	}
	_, err := l.out.Write(l.buf)
	if err != nil {
		panic(err)
	}
}

var (
	outLog   *Logger = nil
	infoLog  *Logger = nil
	warnLog  *Logger = nil
	errorLog *Logger = nil
)

func init() {
	_, err := os.Stat("./log")
	if err != nil {
		_ = os.Mkdir("log", 0777)
	}
	logFile, _ := os.OpenFile("./log/log4u.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	outLog = newLogger(io.MultiWriter(logFile, os.Stdout), "", 0)
	infoLog = newLogger(io.MultiWriter(logFile, os.Stdout), "\u001B[34mINFO\u001B[0m ", LstdFlags|Lmsgprefix|Lshortfile)
	warnLog = newLogger(io.MultiWriter(logFile, os.Stdout), "\u001B[33mWARN\u001B[0m ", LstdFlags|Lmsgprefix|Lshortfile)
	errorLog = newLogger(io.MultiWriter(logFile, os.Stdout), "\u001B[31mERROR\u001B[0m ", LstdFlags|Lmsgprefix|Lshortfile)

	globalLog4u = &Log4u{o: outLog, i: infoLog, w: warnLog, e: errorLog, level: OutLevel, c: make(chan *logInfo, 200)}
	go globalLog4u.outLog()
}

type logInfo struct {
	level LogLevel
	line  int
	file  string
	val   *string
}

var globalLog4u *Log4u = nil

type Log4u struct {
	o     *Logger
	i     *Logger
	w     *Logger
	e     *Logger
	level LogLevel
	c     chan *logInfo
}

func Inject() *Log4u {
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

func getFileAndLine(depth int) (string, int) {
	var file string
	var line int
	var ok bool
	_, file, line, ok = runtime.Caller(depth + 1)
	if !ok {
		file = "???"
		line = 0
	}
	return file, line
}

var panicBytes = []byte("/src/runtime/panic.go")
var otherBytes = []byte(")\n\t")

func getFileAndLineByStack(depth int) (string, int) {

	stack := debug.Stack()
	index := bytes.Index(stack, panicBytes)
	if index == -1 {
		return getFileAndLine(depth + 1)
	}

	stack = stack[index:]
	index = bytes.Index(stack, otherBytes)
	stack = stack[index:]
	stack = stack[3:]
	index = bytes.IndexByte(stack, ' ')
	stack = stack[:index]
	index = bytes.LastIndexByte(stack, ':')

	line, _ := strconv.ParseInt(string(stack[index+1:]), 0, 64)
	return string(stack[:index]), int(line)
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

func (l *Log4u) errCatch() {
	err := recover()
	if err != nil {
		os.Exit(1)
	}
}

func (l *Log4u) outLog() {
	defer l.errCatch()
	for {
		info := <-l.c
		switch info.level {
		case OutLevel:
			l.o.Output(*info.val, info.file, info.line)
		case InfoLevel:
			l.i.Output(*info.val, info.file, info.line)
		case WarnLevel:
			l.w.Output(*info.val, info.file, info.line)
		case ErrorLevel:
			l.e.Output(*info.val, info.file, info.line)
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
	file, line := getFileAndLineByStack(1)
	l.c <- &logInfo{level: InfoLevel, line: line, file: file, val: &str}
}

func (l *Log4u) INFOF(format string, v ...any) {
	if l.level < InfoLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	file, line := getFileAndLineByStack(1)
	l.c <- &logInfo{level: InfoLevel, line: line, file: file, val: &str}
}

func (l *Log4u) WARN(v ...any) {
	if l.level < WarnLevel {
		return
	}
	str := fmt.Sprintln(v...)
	file, line := getFileAndLineByStack(1)
	l.c <- &logInfo{level: WarnLevel, line: line, file: file, val: &str}
}

func (l *Log4u) WARNF(format string, v ...any) {
	if l.level < WarnLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	file, line := getFileAndLineByStack(1)
	l.c <- &logInfo{level: WarnLevel, line: line, file: file, val: &str}
}

func (l *Log4u) ERROR(v ...any) {
	str := fmt.Sprintln(v...)
	file, line := getFileAndLineByStack(1)
	l.c <- &logInfo{level: ErrorLevel, line: line, file: file, val: &str}
}

func (l *Log4u) ERRORF(format string, v ...any) {
	str := fmt.Sprintf(format, v...)
	file, line := getFileAndLineByStack(1)
	l.c <- &logInfo{level: ErrorLevel, line: line, file: file, val: &str}
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
	file, line := getFileAndLineByStack(1)
	globalLog4u.c <- &logInfo{level: InfoLevel, line: line, file: file, val: &str}
}

func INFOF(format string, v ...any) {
	if globalLog4u.level < InfoLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	file, line := getFileAndLineByStack(1)
	globalLog4u.c <- &logInfo{level: InfoLevel, line: line, file: file, val: &str}
}

func WARN(v ...any) {
	if globalLog4u.level < WarnLevel {
		return
	}
	str := fmt.Sprintln(v...)
	file, line := getFileAndLineByStack(1)
	globalLog4u.c <- &logInfo{level: WarnLevel, line: line, file: file, val: &str}
}

func WARNF(format string, v ...any) {
	if globalLog4u.level < WarnLevel {
		return
	}
	str := fmt.Sprintf(format, v...)
	file, line := getFileAndLineByStack(1)
	globalLog4u.c <- &logInfo{level: WarnLevel, line: line, file: file, val: &str}
}

func ERROR(v ...any) {
	str := fmt.Sprintln(v...)
	file, line := getFileAndLineByStack(1)
	globalLog4u.c <- &logInfo{level: ErrorLevel, line: line, file: file, val: &str}
}

func ERRORF(format string, v ...any) {
	str := fmt.Sprintf(format, v...)
	file, line := getFileAndLineByStack(1)
	globalLog4u.c <- &logInfo{level: ErrorLevel, line: line, file: file, val: &str}
}
