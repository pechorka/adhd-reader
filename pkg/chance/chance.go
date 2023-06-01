package chance

import "math/rand"

func Win(percent float64) bool {
	return rand.Float64() < percent
}
