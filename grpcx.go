package grpcx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var (
	ErrConnNotInit    = errors.New("grpc not init")
	ErrConnShutdown   = errors.New("grpc conn shutdown")
	ErrNotFoundStatus = errors.New("grpc parse error status failed")
)

// how to set real ip in nginx
// server {
//     listen  9099  http2;
//     access_log    /var/log/nginx/access-grpc.log;
//     location / {
//         grpc_pass grpc://127.0.0.1:9091;
//         grpc_set_header X-Real-IP $remote_addr;
//     }
// }

// GetRealAddr get real client ip
func GetRealAddr(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	rips := md.Get("x-real-ip")
	if len(rips) == 0 {
		return ""
	}

	return rips[0]
}

// GetPeerAddr get peer addr
func GetPeerAddr(ctx context.Context) string {
	var addr string
	if pr, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := pr.Addr.(*net.TCPAddr); ok {
			addr = tcpAddr.IP.String()
		} else {
			addr = pr.Addr.String()
		}
	}
	return addr
}

func NewServerCreds(cert, key []byte) (credentials.TransportCredentials, error) {
	creds, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{creds}}), nil
}

func NewClientCreds(cert []byte, name string) (credentials.TransportCredentials, error) {
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(cert) {
		return nil, fmt.Errorf("credentials: failed to append certificates")
	}

	return credentials.NewTLS(&tls.Config{ServerName: name, RootCAs: cp}), nil
}

func ParseMethod(full string) (module, call string) {
	s1 := strings.Split(full, "/")
	if l := len(s1); l > 2 {
		call = s1[l-1]

		s2 := strings.Split(s1[l-2], ".")
		if l := len(s2); l > 0 {
			module = s2[l-1]
		}
	}

	if call == "" {
		call = "unknown"
	}

	if module == "" {
		module = "unknown"
	}

	return
}

func CheckConnState(gclient *grpc.ClientConn) error {
	if gclient == nil {
		return ErrConnNotInit
	}

	state := gclient.GetState()
	switch state {
	case connectivity.TransientFailure, connectivity.Shutdown:
		return ErrConnShutdown
	}

	return nil
}

func StatusNotAuth(err interface{}) error {
	out := convs(err, codes.Unauthenticated)
	return status.Errorf(codes.Unauthenticated, out)
}

func StatusUnavailable(err interface{}) error {
	out := convs(err, codes.Unavailable)
	return status.Errorf(codes.Unavailable, out)
}

func StatusNotFound(err interface{}) error {
	out := convs(err, codes.NotFound)
	return status.Errorf(codes.NotFound, out)
}

func StatusExhausted(err interface{}) error {
	out := convs(err, codes.ResourceExhausted)
	return status.Errorf(codes.ResourceExhausted, out)
}

func StatusInternal(err interface{}) error {
	out := convs(err, codes.Internal)
	return status.Errorf(codes.Internal, out)
}

func StatusPermissionDenied(err interface{}) error {
	out := convs(err, codes.PermissionDenied)
	return status.Errorf(codes.PermissionDenied, out)
}

func StatusInvalidArgument(err interface{}) error {
	out := convs(err, codes.InvalidArgument)
	return status.Errorf(codes.InvalidArgument, out)
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
