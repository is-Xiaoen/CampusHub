package breakerx

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/collection"
)

const (
	defaultWindow   = 10 * time.Second
	defaultBuckets  = 40
	defaultRequests = 100
	defaultError    = 0.5
	defaultTimeout  = 60 * time.Second
)

type SREConfig struct {
	Name      string
	Requests  int
	ErrorRate float64
	Timeout   time.Duration
}

// NewSREBreaker creates a breaker with request/error thresholds and open timeout.
func NewSREBreaker(cfg SREConfig) breaker.Breaker {
	requests := cfg.Requests
	if requests <= 0 {
		requests = defaultRequests
	}
	errorRate := cfg.ErrorRate
	if errorRate <= 0 || errorRate > 1 {
		errorRate = defaultError
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &sreBreaker{
		name:      cfg.Name,
		requests:  int64(requests),
		errorRate: errorRate,
		timeout:   timeout,
		window: collection.NewRollingWindow[int64, *collection.Bucket[int64]](
			func() *collection.Bucket[int64] {
				return &collection.Bucket[int64]{}
			},
			defaultBuckets,
			defaultWindow/time.Duration(defaultBuckets),
		),
	}
}

type sreBreaker struct {
	name      string
	requests  int64
	errorRate float64
	timeout   time.Duration
	window    *collection.RollingWindow[int64, *collection.Bucket[int64]]

	mu        sync.Mutex
	openUntil time.Time
}

func (b *sreBreaker) Name() string {
	return b.name
}

func (b *sreBreaker) Allow() (breaker.Promise, error) {
	if b.isOpen() {
		return nil, breaker.ErrServiceUnavailable
	}
	return srePromise{b: b}, nil
}

func (b *sreBreaker) AllowCtx(ctx context.Context) (breaker.Promise, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return b.Allow()
	}
}

func (b *sreBreaker) Do(req func() error) error {
	return b.DoWithAcceptable(req, nil)
}

func (b *sreBreaker) DoCtx(ctx context.Context, req func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return b.Do(req)
	}
}

func (b *sreBreaker) DoWithAcceptable(req func() error, acceptable breaker.Acceptable) error {
	return b.doReq(req, nil, acceptable)
}

func (b *sreBreaker) DoWithAcceptableCtx(ctx context.Context, req func() error, acceptable breaker.Acceptable) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return b.DoWithAcceptable(req, acceptable)
	}
}

func (b *sreBreaker) DoWithFallback(req func() error, fallback breaker.Fallback) error {
	return b.doReq(req, fallback, nil)
}

func (b *sreBreaker) DoWithFallbackCtx(ctx context.Context, req func() error, fallback breaker.Fallback) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return b.DoWithFallback(req, fallback)
	}
}

func (b *sreBreaker) DoWithFallbackAcceptable(
	req func() error,
	fallback breaker.Fallback,
	acceptable breaker.Acceptable,
) error {
	return b.doReq(req, fallback, acceptable)
}

func (b *sreBreaker) DoWithFallbackAcceptableCtx(
	ctx context.Context,
	req func() error,
	fallback breaker.Fallback,
	acceptable breaker.Acceptable,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return b.DoWithFallbackAcceptable(req, fallback, acceptable)
	}
}

func (b *sreBreaker) doReq(req func() error, fallback breaker.Fallback, acceptable breaker.Acceptable) error {
	if acceptable == nil {
		acceptable = func(err error) bool { return err == nil }
	}
	if b.isOpen() {
		if fallback != nil {
			return fallback(breaker.ErrServiceUnavailable)
		}
		return breaker.ErrServiceUnavailable
	}

	defer func() {
		if e := recover(); e != nil {
			b.record(false)
			panic(e)
		}
	}()

	err := req()
	if acceptable(err) {
		b.record(true)
	} else {
		b.record(false)
	}
	return err
}

func (b *sreBreaker) isOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openUntil.IsZero() {
		return false
	}
	if time.Now().Before(b.openUntil) {
		return true
	}
	b.openUntil = time.Time{}
	return false
}

func (b *sreBreaker) record(success bool) {
	if success {
		b.window.Add(0)
	} else {
		b.window.Add(1)
	}
	errors, total := b.history()
	if total < b.requests {
		return
	}
	if float64(errors)/float64(total) >= b.errorRate {
		b.mu.Lock()
		b.openUntil = time.Now().Add(b.timeout)
		b.mu.Unlock()
	}
}

func (b *sreBreaker) history() (errors, total int64) {
	var errSum int64
	var count int64
	b.window.Reduce(func(bucket *collection.Bucket[int64]) {
		errSum += bucket.Sum
		count += bucket.Count
	})
	return errSum, count
}

type srePromise struct {
	b *sreBreaker
}

func (p srePromise) Accept() {
	p.b.record(true)
}

func (p srePromise) Reject(_ string) {
	p.b.record(false)
}
