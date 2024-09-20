package ratelimiter

import (
	"context"
	"log/slog"
	"sync"

	"golang.org/x/time/rate"
)

type Opts struct {
	PerUserLimit int
	GlobalLimit  int
}

type Ratelimiter struct {
	perUserLimit     int
	perUserMu        sync.RWMutex
	perUserRatelimit map[string]*rate.Limiter
	globalRatelimit  *rate.Limiter
}

func NewRatelimiter(opts Opts) *Ratelimiter {
	return &Ratelimiter{
		perUserLimit:     opts.PerUserLimit,
		perUserMu:        sync.RWMutex{},
		perUserRatelimit: make(map[string]*rate.Limiter),
		globalRatelimit:  rate.NewLimiter(rate.Limit(opts.GlobalLimit), opts.GlobalLimit),
	}
}

func (rl *Ratelimiter) Allow(ctx context.Context, user string) bool {
	if err := rl.getOrInitPULimiter(user).Wait(ctx); err != nil {
		slog.WarnContext(ctx, "cancelled while waiting for per user ratelimit quota")
		return false
	}
	if err := rl.globalRatelimit.Wait(ctx); err != nil {
		slog.WarnContext(ctx, "cancelled while waiting for global ratelimit quota")
		return false
	}

	return true
}

func (rl *Ratelimiter) getOrInitPULimiter(user string) *rate.Limiter {
	if limiter := rl.tryGetPULimiter(user); limiter != nil {
		return limiter
	}

	rl.perUserMu.Lock()
	defer rl.perUserMu.Unlock()
	// double check since there is a gap between critical sections.
	if limiter, ok := rl.perUserRatelimit[user]; ok {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.perUserLimit), rl.perUserLimit)
	rl.perUserRatelimit[user] = limiter

	return limiter
}

func (rl *Ratelimiter) tryGetPULimiter(user string) *rate.Limiter {
	rl.perUserMu.RLock()
	defer rl.perUserMu.RUnlock()
	if limiter, ok := rl.perUserRatelimit[user]; ok {
		return limiter
	}
	return nil
}
