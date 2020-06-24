package grpcx

import (
	"fmt"
	"log"
)

var (
	defaultLogger Logger = new(stdLogger)
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
