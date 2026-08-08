package server

import (
	"sync"
	"time"
)

// loginRateLimiter blunts online brute-force attacks by temporarily locking a
// key (client IP + username) after too many consecutive failed logins.
type loginRateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]*loginAttempt
	maxFailures int
	window      time.Duration
	lockout     time.Duration
	now         func() time.Time
}

type loginAttempt struct {
	failures    int
	firstFail   time.Time
	lockedUntil time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		attempts:    make(map[string]*loginAttempt),
		maxFailures: 5,
		window:      10 * time.Minute,
		lockout:     10 * time.Minute,
		now:         time.Now,
	}
}

// checkLocked reports whether the key is currently locked and for how much longer.
func (l *loginRateLimiter) checkLocked(key string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt, ok := l.attempts[key]
	if !ok {
		return 0, false
	}
	now := l.now()
	if now.Before(attempt.lockedUntil) {
		return attempt.lockedUntil.Sub(now), true
	}
	return 0, false
}

// recordFailure registers a failed attempt and locks the key once the failure
// threshold is reached within the window. It returns the lockout duration when
// a lock is newly applied.
func (l *loginRateLimiter) recordFailure(key string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	attempt, ok := l.attempts[key]
	if !ok || now.Sub(attempt.firstFail) > l.window {
		attempt = &loginAttempt{firstFail: now}
		l.attempts[key] = attempt
	}
	attempt.failures++
	l.pruneLocked(now)
	if attempt.failures >= l.maxFailures {
		attempt.lockedUntil = now.Add(l.lockout)
		attempt.failures = 0
		attempt.firstFail = now
		return l.lockout, true
	}
	return 0, false
}

func (l *loginRateLimiter) recordSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

// pruneLocked drops entries that are neither locked nor accumulating, so the
// map stays bounded. Callers must hold the lock.
func (l *loginRateLimiter) pruneLocked(now time.Time) {
	if len(l.attempts) < 1024 {
		return
	}
	for key, attempt := range l.attempts {
		if now.After(attempt.lockedUntil) && now.Sub(attempt.firstFail) > l.window {
			delete(l.attempts, key)
		}
	}
}
