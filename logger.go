package go_logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Level int

func (level Level) Short() string {
	switch level {
	case TRACE:
		return "T"
	case DEBUG:
		return "D"
	case INFO:
		return "I"
	case WARN:
		return "W"
	case ERROR:
		return "E"
	case FATAL:
		return "F"
	default:
		return "?"
	}
}
func (level Level) Long() string {
	switch level {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "?"
	}
}
func (level Level) MarshalJSON() ([]byte, error) {
	return []byte(level.Long()), nil
}

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

type Format int

const (
	PLAIN Format = iota
	JSON
)

var goRoutineNames = make(map[int]string)
var goRoutineNamesMutex = sync.RWMutex{}

func SetGoroutineName(name string) func() {
	id := goroutineId()
	goRoutineNamesMutex.Lock()
	goRoutineNames[id] = name
	goRoutineNamesMutex.Unlock()
	return func() {
		RemoveGoroutineName(id)
	}
}
func RemoveGoroutineName(id int) {
	goRoutineNamesMutex.Lock()
	delete(goRoutineNames, id)
	goRoutineNamesMutex.Unlock()
}

func goroutineName(id int) string {
	goRoutineNamesMutex.RLock()
	name, ok := goRoutineNames[id]
	goRoutineNamesMutex.RUnlock()
	if ok {
		return name
	} else {
		return strconv.Itoa(id)
	}
}

type Logger struct {
	out                    io.Writer
	name                   string
	level                  Level
	format                 Format
	colorized              bool
	panicOnFatal           bool
	maxNameLength          int
	maxGoroutineNameLength int
}

type Event struct {
	Timestamp   time.Time
	GoroutineId string
	Level       Level
	Message     string
	Err         error
}

func NewLogger(name string) *Logger {
	return &Logger{
		out:                    os.Stderr,
		level:                  WARN,
		name:                   name,
		format:                 PLAIN,
		colorized:              false,
		panicOnFatal:           false,
		maxNameLength:          10,
		maxGoroutineNameLength: 10,
	}
}

func (logger *Logger) Out(out io.Writer) *Logger {
	logger.out = out
	return logger
}
func (logger *Logger) Format(format Format) *Logger {
	logger.format = format
	return logger
}
func (logger *Logger) Level(level Level) *Logger {
	logger.level = level
	return logger
}
func (logger *Logger) Colorized(colorized bool) *Logger {
	logger.colorized = colorized
	return logger
}
func (logger *Logger) PanicOnFatal(panicOnFatal bool) *Logger {
	logger.panicOnFatal = panicOnFatal
	return logger
}
func (logger *Logger) MaxNameLength(length int) *Logger {
	logger.maxNameLength = length
	return logger
}
func (logger *Logger) MaxGoroutineNameLength(length int) *Logger {
	logger.maxGoroutineNameLength = length
	return logger
}

func (logger *Logger) log(event *Event) {
	if event.Level >= logger.level {
		switch logger.format {
		case PLAIN:
			logger.logPlain(event)
		case JSON:
			logger.logJson(event)
		}
	}
	if event.Level == FATAL && logger.panicOnFatal {
		panic(event.Err)
	}
}

func (logger *Logger) logPlain(event *Event) {
	sb := strings.Builder{}
	sb.WriteString(event.Timestamp.Format(time.RFC3339))
	sb.WriteString(" -")
	sb.WriteString(event.Level.Short())
	sb.WriteString("- [")
	name := logger.name
	maxNameLength := logger.maxNameLength
	if maxNameLength > 0 {
		name = stringToLength(name, maxNameLength)
	}
	sb.WriteString(name)
	sb.WriteString("] (")
	goId := event.GoroutineId
	maxGoroutineNameLength := logger.maxGoroutineNameLength
	if maxGoroutineNameLength > 0 {
		goId = stringToLength(goId, maxGoroutineNameLength)
	}
	sb.WriteString(goId)
	sb.WriteString(") ")
	sb.WriteString(event.Message)
	if event.Err != nil {
		sb.WriteString(": ")
		sb.WriteString(event.Err.Error())
	}
	sb.WriteByte('\n')
	_, _ = fmt.Fprintf(logger.out, sb.String())
}

func stringToLength(str string, length int) string {
	s := str
	if len(s) > length {
		s = s[:length-3] + "..."
	} else if len(s) < length {
		s = s + strings.Repeat(" ", length-len(s))
	}
	return s
}

func (logger *Logger) logJson(event *Event) {
	sb := strings.Builder{}
	sb.WriteString("{\"timestamo\":\"")
	sb.WriteString(event.Timestamp.Format(time.RFC3339))
	sb.WriteString("\",\"logger\":\"")
	sb.WriteString(logger.name)
	sb.WriteString("\",\"level\":\"")
	sb.WriteString(event.Level.Short())
	sb.WriteString("\",\"goroutineId\":\"")
	sb.WriteString(event.GoroutineId)
	sb.WriteString("\",\"message\":\"")
	message, _ := json.Marshal(event.Message)
	sb.Write(message)
	sb.WriteString("\"")
	if event.Err != nil {
		sb.WriteString(":\"error\":\"")
		err, _ := json.Marshal(event.Err.Error())
		sb.Write(err)
		sb.WriteString("\"")
	}
	sb.WriteString("}\n")
	_, _ = fmt.Fprintf(logger.out, sb.String())
}

func createEvent(level Level, msg string, err error) *Event {
	timestamp := time.Now()
	if msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return &Event{
		Timestamp:   timestamp,
		GoroutineId: goroutineName(goroutineId()),
		Level:       level,
		Message:     msg,
		Err:         err,
	}
}

func goroutineId() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		return -1
	}
	return id
}

func (logger *Logger) Trace(msg string)                  { logger.TraceErr(nil, msg) }
func (logger *Logger) Debug(msg string)                  { logger.DebugErr(nil, msg) }
func (logger *Logger) Info(msg string)                   { logger.InfoErr(nil, msg) }
func (logger *Logger) Warn(msg string)                   { logger.WarnErr(nil, msg) }
func (logger *Logger) Error(msg string)                  { logger.ErrorErr(nil, msg) }
func (logger *Logger) Fatal(msg string)                  { logger.FatalErr(nil, msg) }
func (logger *Logger) TraceErr(err error, msg string)    { logger.log(createEvent(TRACE, msg, err)) }
func (logger *Logger) DebugErr(err error, msg string)    { logger.log(createEvent(DEBUG, msg, err)) }
func (logger *Logger) InfoErr(err error, msg string)     { logger.log(createEvent(INFO, msg, err)) }
func (logger *Logger) WarnErr(err error, msg string)     { logger.log(createEvent(WARN, msg, err)) }
func (logger *Logger) ErrorErr(err error, msg string)    { logger.log(createEvent(ERROR, msg, err)) }
func (logger *Logger) FatalErr(err error, msg string)    { logger.log(createEvent(FATAL, msg, err)) }
func (logger *Logger) Tracef(format string, args ...any) { logger.TraceErrf(nil, format, args...) }
func (logger *Logger) Debugf(format string, args ...any) { logger.DebugErrf(nil, format, args...) }
func (logger *Logger) Infof(format string, args ...any)  { logger.InfoErrf(nil, format, args...) }
func (logger *Logger) Warnf(format string, args ...any)  { logger.WarnErrf(nil, format, args...) }
func (logger *Logger) Errorf(format string, args ...any) { logger.ErrorErrf(nil, format, args...) }
func (logger *Logger) Fatalf(format string, args ...any) { logger.FatalErrf(nil, format, args...) }
func (logger *Logger) TraceErrf(err error, format string, args ...any) {
	logger.log(createEvent(TRACE, fmt.Sprintf(format, args...), err))
}
func (logger *Logger) DebugErrf(err error, format string, args ...any) {
	logger.log(createEvent(DEBUG, fmt.Sprintf(format, args...), err))
}
func (logger *Logger) InfoErrf(err error, format string, args ...any) {
	logger.log(createEvent(INFO, fmt.Sprintf(format, args...), err))
}
func (logger *Logger) WarnErrf(err error, format string, args ...any) {
	logger.log(createEvent(WARN, fmt.Sprintf(format, args...), err))
}
func (logger *Logger) ErrorErrf(err error, format string, args ...any) {
	logger.log(createEvent(ERROR, fmt.Sprintf(format, args...), err))
}
func (logger *Logger) FatalErrf(err error, format string, args ...any) {
	logger.log(createEvent(FATAL, fmt.Sprintf(format, args...), err))
}
func (logger *Logger) IsTrace() bool { return logger.level <= TRACE }
func (logger *Logger) IsDebug() bool { return logger.level <= DEBUG }
func (logger *Logger) IsInfo() bool  { return logger.level <= INFO }
func (logger *Logger) IsWarn() bool  { return logger.level <= WARN }
func (logger *Logger) IsError() bool { return logger.level <= ERROR }
func (logger *Logger) IsFatal() bool { return logger.level <= FATAL }
