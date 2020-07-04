package grpcx

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strings"
)

var (
	defaultLogger Logger = new(stdLogger)
	dunno                = []byte("???")
	centerDot            = []byte("·")
	dot                  = []byte(".")
	slash                = []byte("/")
)

type Logger interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
}

func SetLogger(logger Logger) {
	defaultLogger = logger
}

type stdLogger struct{}

func (s *stdLogger) Infof(format string, args ...interface{}) {
	log.Println(fmt.Sprintf(format, args...))
}

func (s *stdLogger) Errorf(format string, args ...interface{}) {
	log.Println(fmt.Sprintf(format, args...))
}

// refer https://github.com/gin-gonic/gin/blob/5e40c1d49c/recovery.go

func getStack(skip int) string {
	buf := new(bytes.Buffer) // the returned data
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		_, _ = fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		_, _ = fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.String()
}

// source code
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function name
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}

	name := []byte(fn.Name())
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

// GetCaller get filename, line, fucntion name
func GetCaller(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0, ""
	}

	var (
		n        = 0
		funcName string
	)

	// get package name
	for i := len(file) - 1; i > 0; i-- {
		if file[i] != '/' {
			continue
		}
		n++
		if n >= 2 {
			file = file[i+1:]
			break
		}
	}

	fnpc := runtime.FuncForPC(pc)

	if fnpc != nil {
		fnNameStr := fnpc.Name()
		parts := strings.Split(fnNameStr, ".")
		funcName = parts[len(parts)-1]
	}

	return file, line, funcName
}
