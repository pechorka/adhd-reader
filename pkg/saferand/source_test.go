package saferand_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/pechorka/adhd-reader/pkg/saferand"
)

func TestSource(t *testing.T) {
	t.Run("test deterministic", func(t *testing.T) {
		seed := time.Now().Unix()
		r1, r2 := rand.New(saferand.NewSource(seed)), rand.New(saferand.NewSource(seed))
		for i := 0; i < 100; i++ {
			n1, n2 := r1.Int63(), r2.Int63()
			if n1 != n2 {
				t.Fatalf("expected values to match seeded with same value, %d != %d", n1, n2)
			}
		}
	})

	t.Run("test race", func(t *testing.T) {
		r := rand.New(saferand.NewSource(time.Now().Unix()))
		wg := sync.WaitGroup{}
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					_ = r.Int()
				}
			}()
		}
		wg.Wait()
	})
}
