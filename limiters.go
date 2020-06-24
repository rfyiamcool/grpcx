package grpcx

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	DefaultRateLimiter = NewRateLimiterPool(50, 100)

	// todo: clean
	defaultCleanInterval = 6 * time.Hour
)

type RateLimiterPool struct {
	r             rate.Limit
	b             int
	limiters      map[string]*rate.Limiter
	cleanInterval time.Duration

	sync.RWMutex
}

func NewRateLimiterPool(r rate.Limit, b int) *RateLimiterPool {
	l := &RateLimiterPool{
		r:        r,
		b:        b,
		limiters: make(map[string]*rate.Limiter),
	}

	return l
}

func (l *RateLimiterPool) AddLimiter(tag string) *rate.Limiter {
	l.Lock()
	defer l.Unlock()

	return l.addLimiter(tag)
}

func (l *RateLimiterPool) addLimiter(tag string) *rate.Limiter {
	limiter := rate.NewLimiter(l.r, l.b)
	l.limiters[tag] = limiter

	return limiter
}

func (l *RateLimiterPool) GetLimiter(tag string) *rate.Limiter {
	l.Lock()
	defer l.Unlock()

	limiter, exists := l.limiters[tag]
	if !exists {
		return l.addLimiter(tag)
	}

	return limiter
}

func (l *RateLimiterPool) Allow(tag string) bool {
	limiter := l.GetLimiter(tag)
	return limiter.Allow()
}

func (l *RateLimiterPool) Wait(ctx context.Context, tag string) error {
	limiter := l.GetLimiter(tag)
	return limiter.Wait(ctx)
}
