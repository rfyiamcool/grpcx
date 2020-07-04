package grpcx

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errNotFound = errors.New("not found service method")

// StreamInterceptorChain returns stream interceptors chain.
func StreamInterceptorChain(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	build := func(c grpc.StreamServerInterceptor, n grpc.StreamHandler, info *grpc.StreamServerInfo) grpc.StreamHandler {
		return func(srv interface{}, stream grpc.ServerStream) error {
			return c(srv, stream, info, n)
		}
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = build(interceptors[i], chain, info)
		}
		return chain(srv, stream)
	}
}

// UnaryInterceptorChain returns interceptors chain.
func UnaryInterceptorChain(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	build := func(c grpc.UnaryServerInterceptor, n grpc.UnaryHandler, info *grpc.UnaryServerInfo) grpc.UnaryHandler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			return c(ctx, req, info, n)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = build(interceptors[i], chain, info)
		}
		return chain(ctx, req)
	}
}

// RecoveryStreamServerInterceptor recover interceptor for grpc
func RecoveryStreamServerInterceptor(role string) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				defaultLogger.Errorf("[%s] grpc stream %s panic recovery, err: %v, stack: %s", role, info.FullMethod, rec, getStack(2))
				err = StatusInternal(rec)
			}
		}()
		return handler(srv, stream)
	}
}

// RecoveryUnaryServerInterceptor recover interceptor for grpc
func RecoveryUnaryServerInterceptor(role string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				defaultLogger.Errorf("[%s] grpc unary %s panic recovery, err: %v, stack: %s", role, info.FullMethod, rec, getStack(2))
				err = StatusInternal(rec)
			}
		}()
		resp, err = handler(ctx, req)
		return resp, err
	}
}

func LoggerUnaryInterceptor(role string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()

		defaultLogger.Infof("[%s] begin grpc unary request %s", role, info.FullMethod)
		resp, err = handler(ctx, req)
		defaultLogger.Infof("[%s] finish grpc unary request %s, cost: %v", role, info.FullMethod, time.Since(start).String())

		return resp, err
	}
}

// IPAllowUnaryInterceptor allow ips only
func IPAllowUnaryInterceptor(allowIps []string) grpc.UnaryServerInterceptor {
	allowMap := map[string]bool{}
	for _, ip := range allowIps {
		allowMap[ip] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		peer := GetRealAddr(ctx)
		if peer == "" {
			peer = GetPeerAddr(ctx)
		}

		if _, ok := allowMap[peer]; ok {
			resp, err = handler(ctx, req)
			return resp, err
		}

		defaultLogger.Errorf("host [%s] request [%s] can't allow request", peer, info.FullMethod)
		return nil, status.Errorf(codes.ResourceExhausted, "host [%s] request [%s] can not allow request", peer, info.FullMethod)
	}
}

// IPDenyUnaryInterceptor deny black ip list only
func IPDenyUnaryInterceptor(denyIPs []string) grpc.UnaryServerInterceptor {
	denyMap := map[string]bool{}
	for _, ip := range denyIPs {
		denyMap[ip] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		peer := GetRealAddr(ctx)
		if peer == "" {
			peer = GetPeerAddr(ctx)
		}

		if _, ok := denyMap[peer]; ok {
			defaultLogger.Errorf("host [%s] request [%s] can't allow request", peer, info.FullMethod)
			return nil, status.Errorf(codes.ResourceExhausted, "host [%s] request [%s] can not allow request", peer, info.FullMethod)
		}

		resp, err = handler(ctx, req)
		return resp, err
	}
}

func IPRateLimiterUnaryInterceptor(limiter *RateLimiterPool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		peer := GetPeerAddr(ctx)
		if !limiter.Allow(peer) {
			defaultLogger.Errorf("host [%s] request [%s] is rejected by ip ratelimiter", peer, info.FullMethod)
			return nil, status.Errorf(codes.ResourceExhausted, "host [%s] request [%s] is rejected by ratelimiter", peer, info.FullMethod)
		}

		resp, err = handler(ctx, req)
		return resp, err
	}
}

func MethodRateLimiterUnaryInterceptor(limiter *RateLimiterPool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		peer := GetPeerAddr(ctx)
		if !limiter.Allow(info.FullMethod) {
			defaultLogger.Errorf("host [%s] request [%s] is rejected by method ratelimiter", peer, info.FullMethod)
			return nil, status.Errorf(codes.ResourceExhausted, "host [%s] request [%s] is rejected by ratelimiter", peer, info.FullMethod)
		}

		resp, err = handler(ctx, req)
		return resp, err
	}
}

func UnknowServiceHandler() grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		fullMethod, ok := grpc.MethodFromServerStream(stream)
		if ok {
			defaultLogger.Errorf("not found service method")
		} else {
			defaultLogger.Errorf("not found service method %s", fullMethod)
		}
		return errNotFound
	}
}

func TryUnaryHandler(ctx context.Context, req interface{}, handler grpc.UnaryHandler) (resp interface{}, err error, paniced bool) {
	defer func() {
		r := recover()
		if r != nil {
			resp = nil
			paniced = true
			err = errors.Errorf("throw %+v", r)
		}
	}()

	resp, err = handler(ctx, req)
	return resp, err, paniced
}
