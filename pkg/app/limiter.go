package app

import (
	"golang.org/x/time/rate"
	"net/http"
	"sync"
)

// RemoteAddrRateLimiter .
type RemoteAddrRateLimiter struct {
	RemoteAddrs map[string]*rate.Limiter
	mu          *sync.RWMutex
	r           rate.Limit
	b           int
}

// NewRemoteAddrRateLimiter .
func NewRemoteAddrRateLimiter(r rate.Limit, b int) *RemoteAddrRateLimiter {
	i := &RemoteAddrRateLimiter{
		RemoteAddrs: make(map[string]*rate.Limiter),
		mu:          &sync.RWMutex{},
		r:           r,
		b:           b,
	}

	return i
}

// AddRemoteAddr creates a new rate limiter and adds it to the RemoteAddrs map,
// using the RemoteAddr address as the key
func (i *RemoteAddrRateLimiter) AddRemoteAddr(RemoteAddr string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter := rate.NewLimiter(i.r, i.b)

	i.RemoteAddrs[RemoteAddr] = limiter

	return limiter
}

// GetLimiter returns the rate limiter for the provided RemoteAddr address if it exists.
// Otherwise calls AddRemoteAddr to add RemoteAddr address to the map
func (i *RemoteAddrRateLimiter) GetLimiter(RemoteAddr string) *rate.Limiter {
	i.mu.Lock()
	limiter, exists := i.RemoteAddrs[RemoteAddr]

	if !exists {
		i.mu.Unlock()
		return i.AddRemoteAddr(RemoteAddr)
	}

	i.mu.Unlock()

	return limiter
}

func (i *RemoteAddrRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := i.GetLimiter(r.RemoteAddr)
		if !limiter.Allow() {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
