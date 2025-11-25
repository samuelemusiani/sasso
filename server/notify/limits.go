// As real external systems are used for notifications (for example email and
// telegram), we cannot accept to have bugs that flood these systems.
// So we implement rate limiting for ourselves.

package notify

import (
	"sync"
	"time"
)

// We implement a simple token bucket rate limiter
// https://en.wikipedia.org/wiki/Token_bucket

type bucketLimiter struct {
	rate       float64 // Rate of token generation per second
	capacity   uint64  // Maximum capacity of the bucket
	tokens     uint64  // Current number of tokens in the bucket
	lastUpdate time.Time
	mu         sync.Mutex
}

func newBucketLimiter(rate float64, capacity uint64) *bucketLimiter {
	return &bucketLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastUpdate: time.Now(),
	}
}

func (b *bucketLimiter) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()

	// Refill tokens based on elapsed time
	newTokens := uint64(elapsed * b.rate)
	if newTokens > 0 {
		b.tokens += newTokens
		if b.tokens > b.capacity {
			b.tokens = b.capacity
		}
		b.lastUpdate = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}
