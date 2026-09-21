package presenter

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SecurityMiddleware adds essential production HTTP security and cache control headers.
func SecurityMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Clickjacking protection
			w.Header().Set("X-Frame-Options", "DENY")
			// MIME sniffing protection
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Referrer Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// Disable legacy XSS filter behavior that introduces vulnerabilities
			w.Header().Set("X-XSS-Protection", "0")

			// For static assets (/static/), allow caching. For all application routes, prevent intermediary caching.
			if strings.HasPrefix(r.URL.Path, "/static/") {
				w.Header().Set("Cache-Control", "public, max-age=86400")
			} else {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
				w.Header().Set("Pragma", "no-cache")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimiter tracks rate limiters by client IP with automatic eviction of stale clients.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientLimiter
	r       rate.Limit
	b       int
	ttl     time.Duration
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter creates a thread-safe rate limiter partitioned by IP.
// r: sustained requests per second. b: burst capacity. ttl: cleanup interval.
func NewIPRateLimiter(r rate.Limit, b int, ttl time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		clients: make(map[string]*clientLimiter),
		r:       r,
		b:       b,
		ttl:     ttl,
	}

	// Periodic cleanup of inactive IPs to avoid memory leaks
	go func() {
		ticker := time.NewTicker(ttl)
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, exists := rl.clients[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.r, rl.b)
		rl.clients[ip] = &clientLimiter{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	c.lastSeen = time.Now()
	return c.limiter
}

func (rl *IPRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.ttl)
	for ip, c := range rl.clients {
		if c.lastSeen.Before(cutoff) {
			delete(rl.clients, ip)
		}
	}
}

// LimitMiddleware wraps an http.Handler with rate limiting based on client IP.
func (rl *IPRateLimiter) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			w.Header().Set("Retry-After", "10")
			http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetClientIP resolves the effective IP address of the client, checking standard proxy headers.
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(parts[0])
		if clientIP != "" {
			return clientIP
		}
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// IsHTTPS returns true if the connection is encrypted or if a reverse proxy reports HTTPS.
func IsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	return strings.EqualFold(proto, "https")
}
