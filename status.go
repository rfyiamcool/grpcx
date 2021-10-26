package grpcx

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Wrap401(format string, a ...interface{}) error {
	return StatusNotAuth(format, a...)
}

func StatusNotAuth(format string, a ...interface{}) error {
	return status.Errorf(codes.Unauthenticated, fmt.Sprintf(format, a...))
}

func Wrap503(format string, a ...interface{}) error {
	return StatusUnavailable(format, a...)
}

func StatusUnavailable(format string, a ...interface{}) error {
	return status.Errorf(codes.Unavailable, fmt.Sprintf(format, a...))
}

func Wrap404(format string, a ...interface{}) error {
	return StatusNotFound(format, a...)
}

func StatusNotFound(format string, a ...interface{}) error {
	return status.Errorf(codes.NotFound, fmt.Sprintf(format, a...))
}

func StatusExhausted(format string, a ...interface{}) error {
	return status.Errorf(codes.ResourceExhausted, fmt.Sprintf(format, a...))
}

func Wrap500(format string, a ...interface{}) error {
	return StatusInternal(format, a...)
}

func StatusInternal(format string, a ...interface{}) error {
	return status.Errorf(codes.Internal, fmt.Sprintf(format, a...))
}

func Wrap403(format string, a ...interface{}) error {
	return StatusPermissionDenied(format, a...)
}

func StatusPermissionDenied(format string, a ...interface{}) error {
	return status.Errorf(codes.PermissionDenied, fmt.Sprintf(format, a...))
}

func Wrap400(format string, a ...interface{}) error {
	return StatusInvalidArgument(format, a...)
}

func StatusInvalidArgument(format string, a ...interface{}) error {
	return status.Errorf(codes.InvalidArgument, fmt.Sprintf(format, a...))
}

func convs(err interface{}, cm codes.Code) string {
	var out string

	if err == nil {
		return cm.String()
	}

	switch err.(type) {
	case error:
		out = err.(error).Error()
	case string:
		out = err.(string)
	case []byte:
		out = string(err.([]byte))
	default:
		// for recovery
		out = fmt.Sprintf("%v", err)
	}

	return out
}

func IsErrorNotAuth(err error) bool {
	e, ok := status.FromError(err)
	if !ok {
		return false
	}

	if e.Code() == codes.Unauthenticated {
		return true
	}
	return false
}

func IsErrorInvalidArgument(err error) bool {
	e, ok := status.FromError(err)
	if !ok {
		return false
	}

	if e.Code() == codes.InvalidArgument {
		return true
	}
	return false
}

func IsErrorInternal(err error) bool {
	e, ok := status.FromError(err)
	if !ok {
		return false
	}

	if e.Code() == codes.Internal {
		return true
	}
	return false
}

func IsErrorPermissionDenied(err error) bool {
	e, ok := status.FromError(err)
	if !ok {
		return false
	}

	if e.Code() == codes.PermissionDenied {
		return true
	}
	return false
}

func IsErrorNotFound(err error) bool {
	e, ok := status.FromError(err)
	if !ok {
		return false
	}

	if e.Code() == codes.NotFound {
		return true
	}
	return false
}

func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	e, ok := status.FromError(err)
	if !ok {
		return err.Error()
	}

	return e.Message()
}

func IsError(serr, derr error) bool {
	e, ok := status.FromError(serr)
	if !ok {
		return false
	}

	if e.Message() == derr.Error() {
		return true
	}
	return false
}

func MatchError(gerr, derr error) bool {
	if gerr == nil && derr == nil {
		return true
	}

	if gerr == nil || derr == nil {
		return false
	}

	e, ok := status.FromError(gerr)
	if !ok {
		return false
	}

	if strings.Contains(e.Message(), derr.Error()) {
		return true
	}

	return false
}
