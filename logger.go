package go_logger

import (
	"encoding/json"
	"fmt"
	"golang.org/x/term"
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

type Logger struct {
	out                    io.Writer
	name                   string
	level                  Level
	format                 Format
	colorizedSet           bool
	colors                 colors
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
		colorizedSet:           false,
		colors:                 colorsOn,
		panicOnFatal:           false,
		maxNameLength:          10,
		maxGoroutineNameLength: 10,
	}
}

func (logger *Logger) Out(out io.Writer) *Logger {
	logger.out = out
	if !logger.colorizedSet {
		if f, ok := out.(*os.File); ok {
			if term.IsTerminal(int(f.Fd())) {
				logger.colors = colorsOn
			} else {
				logger.colors = colorsOff
			}
		}
	}
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
	logger.colorizedSet = true
	if colorized {
		logger.colors = colorsOn
	} else {
		logger.colors = colorsOff
	}
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

func (logger *Logger) Trace(msg string) { logger.log(createEvent(TRACE, msg, nil)) }
func (logger *Logger) Debug(msg string) { logger.log(createEvent(DEBUG, msg, nil)) }
func (logger *Logger) Info(msg string)  { logger.log(createEvent(INFO, msg, nil)) }
func (logger *Logger) Warn(msg string)  { logger.log(createEvent(WARN, msg, nil)) }
func (logger *Logger) Error(msg string) { logger.log(createEvent(ERROR, msg, nil)) }
func (logger *Logger) Fatal(msg string) { logger.log(createEvent(FATAL, msg, nil)) }
func (logger *Logger) Tracef(format string, args ...any) {
	logger.log(createEvent(TRACE, fmt.Sprintf(format, args...), nil))
}
func (logger *Logger) Debugf(format string, args ...any) {
	logger.log(createEvent(DEBUG, fmt.Sprintf(format, args...), nil))
}
func (logger *Logger) Infof(format string, args ...any) {
	logger.log(createEvent(INFO, fmt.Sprintf(format, args...), nil))
}
func (logger *Logger) Warnf(format string, args ...any) {
	logger.log(createEvent(WARN, fmt.Sprintf(format, args...), nil))
}
func (logger *Logger) Errorf(format string, args ...any) {
	logger.log(createEvent(ERROR, fmt.Sprintf(format, args...), nil))
}
func (logger *Logger) Fatalf(format string, args ...any) {
	logger.log(createEvent(FATAL, fmt.Sprintf(format, args...), nil))
}

func (logger *Logger) TraceErr(err error, msg string) {
	if err != nil {
		logger.log(createEvent(TRACE, msg, err))
	}
}
func (logger *Logger) DebugErr(err error, msg string) {
	if err != nil {
		logger.log(createEvent(DEBUG, msg, err))
	}
}
func (logger *Logger) InfoErr(err error, msg string) {
	if err != nil {
		logger.log(createEvent(INFO, msg, err))
	}
}
func (logger *Logger) WarnErr(err error, msg string) {
	if err != nil {
		logger.log(createEvent(WARN, msg, err))
	}
}
func (logger *Logger) ErrorErr(err error, msg string) {
	if err != nil {
		logger.log(createEvent(ERROR, msg, err))
	}
}
func (logger *Logger) FatalErr(err error, msg string) {
	if err != nil {
		logger.log(createEvent(FATAL, msg, err))
	}
}
func (logger *Logger) TraceErrf(err error, format string, args ...any) {
	if err != nil {
		logger.log(createEvent(TRACE, fmt.Sprintf(format, args...), err))
	}
}
func (logger *Logger) DebugErrf(err error, format string, args ...any) {
	if err != nil {
		logger.log(createEvent(DEBUG, fmt.Sprintf(format, args...), err))
	}
}
func (logger *Logger) InfoErrf(err error, format string, args ...any) {
	if err != nil {
		logger.log(createEvent(INFO, fmt.Sprintf(format, args...), err))
	}
}
func (logger *Logger) WarnErrf(err error, format string, args ...any) {
	if err != nil {
		logger.log(createEvent(WARN, fmt.Sprintf(format, args...), err))
	}
}
func (logger *Logger) ErrorErrf(err error, format string, args ...any) {
	if err != nil {
		logger.log(createEvent(ERROR, fmt.Sprintf(format, args...), err))
	}
}
func (logger *Logger) FatalErrf(err error, format string, args ...any) {
	if err != nil {
		logger.log(createEvent(FATAL, fmt.Sprintf(format, args...), err))
	}
}
func (logger *Logger) IsTrace() bool { return logger.level <= TRACE }
func (logger *Logger) IsDebug() bool { return logger.level <= DEBUG }
func (logger *Logger) IsInfo() bool  { return logger.level <= INFO }
func (logger *Logger) IsWarn() bool  { return logger.level <= WARN }
func (logger *Logger) IsError() bool { return logger.level <= ERROR }
func (logger *Logger) IsFatal() bool { return logger.level <= FATAL }

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

var goRoutineNames = make(map[int]string)
var goRoutineNamesMutex = sync.RWMutex{}

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
	sb.WriteString(logger.colors.cGREY)
	sb.WriteString(event.Timestamp.Format(time.RFC3339))
	sb.WriteString(levelColored(logger, event.Level))
	sb.WriteString(" -")
	sb.WriteString(event.Level.Short())
	sb.WriteString("-")
	sb.WriteString(logger.colors.cGREY)
	sb.WriteString(" [")
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
	sb.WriteString(logger.colors.cWHITE)
	sb.WriteString(event.Message)
	if event.Err != nil {
		sb.WriteString(": ")
		sb.WriteString(event.Err.Error())
	}
	sb.WriteString(logger.colors.cEND)
	sb.WriteByte('\n')
	_, _ = fmt.Fprintf(logger.out, sb.String())
}

func levelColored(logger *Logger, level Level) string {
	switch level {
	case TRACE:
		return logger.colors.cBLUE
	case DEBUG:
		return logger.colors.cBLUE2
	case INFO:
		return logger.colors.cYELLOW
	case WARN:
		return logger.colors.cYELLOW2
	case ERROR:
		return logger.colors.cRED
	case FATAL:
		return logger.colors.cRED2
	default:
		return ""
	}
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

type colors struct {
	cEND       string
	cBOLD      string
	cITALIC    string
	cURL       string
	cBLINK     string
	cBLINK2    string
	cSELECTED  string
	cBLACK     string
	cRED       string
	cGREEN     string
	cYELLOW    string
	cBLUE      string
	cVIOLET    string
	cBEIGE     string
	cWHITE     string
	cBLACKBG   string
	cREDBG     string
	cGREENBG   string
	cYELLOWBG  string
	cBLUEBG    string
	cVIOLETBG  string
	cBEIGEBG   string
	cWHITEBG   string
	cGREY      string
	cRED2      string
	cGREEN2    string
	cYELLOW2   string
	cBLUE2     string
	cVIOLET2   string
	cBEIGE2    string
	cWHITE2    string
	cGREYBG    string
	cREDBG2    string
	cGREENBG2  string
	cYELLOWBG2 string
	cBLUEBG2   string
	cVIOLETBG2 string
	cBEIGEBG2  string
	cWHITEBG2  string
}

var colorsOff = colors{
	cEND:       "",
	cBOLD:      "",
	cITALIC:    "",
	cURL:       "",
	cBLINK:     "",
	cBLINK2:    "",
	cSELECTED:  "",
	cBLACK:     "",
	cRED:       "",
	cGREEN:     "",
	cYELLOW:    "",
	cBLUE:      "",
	cVIOLET:    "",
	cBEIGE:     "",
	cWHITE:     "",
	cBLACKBG:   "",
	cREDBG:     "",
	cGREENBG:   "",
	cYELLOWBG:  "",
	cBLUEBG:    "",
	cVIOLETBG:  "",
	cBEIGEBG:   "",
	cWHITEBG:   "",
	cGREY:      "",
	cRED2:      "",
	cGREEN2:    "",
	cYELLOW2:   "",
	cBLUE2:     "",
	cVIOLET2:   "",
	cBEIGE2:    "",
	cWHITE2:    "",
	cGREYBG:    "",
	cREDBG2:    "",
	cGREENBG2:  "",
	cYELLOWBG2: "",
	cBLUEBG2:   "",
	cVIOLETBG2: "",
	cBEIGEBG2:  "",
	cWHITEBG2:  "",
}
var colorsOn = colors{
	cEND:       "\033[0m",
	cBOLD:      "\033[1m",
	cITALIC:    "\033[3m",
	cURL:       "\033[4m",
	cBLINK:     "\033[5m",
	cBLINK2:    "\033[6m",
	cSELECTED:  "\033[7m",
	cBLACK:     "\033[30m",
	cRED:       "\033[31m",
	cGREEN:     "\033[32m",
	cYELLOW:    "\033[33m",
	cBLUE:      "\033[34m",
	cVIOLET:    "\033[35m",
	cBEIGE:     "\033[36m",
	cWHITE:     "\033[37m",
	cBLACKBG:   "\033[40m",
	cREDBG:     "\033[41m",
	cGREENBG:   "\033[42m",
	cYELLOWBG:  "\033[43m",
	cBLUEBG:    "\033[44m",
	cVIOLETBG:  "\033[45m",
	cBEIGEBG:   "\033[46m",
	cWHITEBG:   "\033[47m",
	cGREY:      "\033[90m",
	cRED2:      "\033[91m",
	cGREEN2:    "\033[92m",
	cYELLOW2:   "\033[93m",
	cBLUE2:     "\033[94m",
	cVIOLET2:   "\033[95m",
	cBEIGE2:    "\033[96m",
	cWHITE2:    "\033[97m",
	cGREYBG:    "\033[100m",
	cREDBG2:    "\033[101m",
	cGREENBG2:  "\033[102m",
	cYELLOWBG2: "\033[103m",
	cBLUEBG2:   "\033[104m",
	cVIOLETBG2: "\033[105m",
	cBEIGEBG2:  "\033[106m",
	cWHITEBG2:  "\033[107m",
}
