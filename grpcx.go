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
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
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
