// Package middleware provides HTTP middleware for the flowrunner API.
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
		rateLimiter:    NewRateLimiter(5, 60*time.Second), // 5 failed attempts per minute
	}
}

// Authenticate is middleware that authenticates requests
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get authentication credentials
		username, password, hasBasicAuth := r.BasicAuth()
		authHeader := r.Header.Get("Authorization")

		var accountID string
		var err error

		// Try Basic Authentication
		if hasBasicAuth {
			// Check rate limiting for this username
			if m.rateLimiter.IsLimited(username) {
				http.Error(w, "Too many authentication attempts, please try again later", http.StatusTooManyRequests)
				return
			}

			accountID, err = m.accountService.Authenticate(username, password)
			if err != nil {
				// Record failed attempt
				m.rateLimiter.RecordAttempt(username)

				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}
		} else if strings.HasPrefix(authHeader, "Bearer ") {
			// Try Bearer Token Authentication
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Check rate limiting for this token (first 8 chars as ID)
			tokenID := token
			if len(token) > 8 {
				tokenID = token[:8]
			}

			if m.rateLimiter.IsLimited(tokenID) {
				http.Error(w, "Too many authentication attempts, please try again later", http.StatusTooManyRequests)
				return
			}

			accountID, err = m.accountService.ValidateToken(token)
			if err != nil {
				// Record failed attempt
				m.rateLimiter.RecordAttempt(tokenID)

				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
		} else {
			// No authentication provided
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Add account ID to context
		ctx := context.WithValue(r.Context(), AccountIDKey, accountID)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetAccountID retrieves the account ID from the request context
func GetAccountID(r *http.Request) (string, bool) {
	accountID, ok := r.Context().Value(AccountIDKey).(string)
	return accountID, ok
}

// RequireAccountID is middleware that ensures an account ID is present in the context
func RequireAccountID(next http.Handler) http.Handler {
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
	attempts     map[string][]time.Time
	maxAttempts  int
	windowPeriod time.Duration
	mu           sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxAttempts int, windowPeriod time.Duration) *RateLimiter {
	return &RateLimiter{
		attempts:     make(map[string][]time.Time),
		maxAttempts:  maxAttempts,
		windowPeriod: windowPeriod,
	}
}

// RecordAttempt records an authentication attempt
func (rl *RateLimiter) RecordAttempt(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up old attempts
	rl.cleanupOldAttempts(key, now)

	// Record new attempt
	rl.attempts[key] = append(rl.attempts[key], now)
}

// IsLimited checks if a key is rate limited
func (rl *RateLimiter) IsLimited(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up old attempts
	rl.cleanupOldAttempts(key, now)

	// Check if number of attempts exceeds the limit
	return len(rl.attempts[key]) >= rl.maxAttempts
}

// cleanupOldAttempts removes attempts outside the window period
func (rl *RateLimiter) cleanupOldAttempts(key string, now time.Time) {
	cutoff := now.Add(-rl.windowPeriod)
	attempts := rl.attempts[key]

	// Find the index of the first attempt within the window
	i := 0
	for ; i < len(attempts); i++ {
		if attempts[i].After(cutoff) {
			break
		}
	}

	// Remove attempts outside the window
	if i > 0 {
		if i < len(attempts) {
			rl.attempts[key] = attempts[i:]
		} else {
			rl.attempts[key] = nil
		}
	}
}
