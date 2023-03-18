package saferand

import (
	"math/rand"
	"sync"
)

type source struct {
	mu  sync.Mutex
	src rand.Source
}

// NewSource returns a rand.Source safe for concurrent use.
func NewSource(seed int64) rand.Source {
	return &source{src: rand.NewSource(seed)}
}

// Seed uses the provided seed value to initialize the generator to a deterministic state.
func (s *source) Seed(seed int64) {
	s.mu.Lock()
	s.src.Seed(seed)
	s.mu.Unlock()
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func (s *source) Int63() int64 {
	s.mu.Lock()
	n := s.src.Int63()
	s.mu.Unlock()
	return n
}
