// Package middleware provides HTTP middleware for flowrunner.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tcmartin/flowrunner/pkg/auth"
)

// Key type for context values
type contextKey string

// Context keys
const (
	AccountIDKey contextKey = "account_id"
)

// AuthMiddleware provides authentication middleware for HTTP handlers
type AuthMiddleware struct {
	accountService auth.AccountService
	rateLimiter    *RateLimiter
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(accountService auth.AccountService) *AuthMiddleware {
	return &AuthMiddleware{
		accountService: accountService,
		rateLimiter:    NewRateLimiter(100, time.Minute), // 100 attempts per minute
	}
}

// Authenticate is middleware that authenticates requests
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for OPTIONS requests (CORS preflight)
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check rate limiting
		clientIP := r.RemoteAddr
		if m.rateLimiter.IsLimited(clientIP) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		var accountID string
		var err error

		// Handle different auth methods
		if strings.HasPrefix(authHeader, "Bearer ") {
			// Bearer token authentication (JWT or API token)
			token := strings.TrimPrefix(authHeader, "Bearer ")
			accountID, err = m.accountService.ValidateToken(token)
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// Basic authentication
			username, password, ok := r.BasicAuth()
			if !ok {
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}
			accountID, err = m.accountService.Authenticate(username, password)
		} else {
			http.Error(w, "Unsupported authentication method", http.StatusUnauthorized)
			return
		}

		// Check authentication result
		if err != nil {
			m.rateLimiter.Record(clientIP)
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		// Add account ID to context
		ctx := context.WithValue(r.Context(), AccountIDKey, accountID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAccountID retrieves the account ID from the request context
func GetAccountID(r *http.Request) (string, bool) {
	accountID, ok := r.Context().Value(AccountIDKey).(string)
	return accountID, ok
}

// RequireAccount is middleware that ensures an account ID is present in the context
func RequireAccount(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := GetAccountID(r); !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimiter implements a simple rate limiting mechanism
type RateLimiter struct {
	attempts   map[string][]time.Time
	limit      int
	window     time.Duration
	mu         sync.Mutex
	cleanupInt time.Duration
	lastClean  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts:   make(map[string][]time.Time),
		limit:      limit,
		window:     window,
		cleanupInt: time.Minute * 5,
		lastClean:  time.Now(),
	}
}

// IsLimited checks if a client is rate limited
func (r *RateLimiter) IsLimited(clientID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up old entries periodically
	if time.Since(r.lastClean) > r.cleanupInt {
		r.cleanup()
		r.lastClean = time.Now()
	}

	// Get attempts for this client
	attempts := r.attempts[clientID]
	if len(attempts) == 0 {
		return false
	}

	// Count attempts within the window
	cutoff := time.Now().Add(-r.window)
	count := 0
	for _, t := range attempts {
		if t.After(cutoff) {
			count++
		}
	}

	return count >= r.limit
}

// Record records an authentication attempt
func (r *RateLimiter) Record(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempts[clientID] = append(r.attempts[clientID], time.Now())
}

// cleanup removes old entries
func (r *RateLimiter) cleanup() {
	cutoff := time.Now().Add(-r.window)
	for clientID, attempts := range r.attempts {
		var valid []time.Time
		for _, t := range attempts {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) > 0 {
			r.attempts[clientID] = valid
		} else {
			delete(r.attempts, clientID)
		}
	}
}
