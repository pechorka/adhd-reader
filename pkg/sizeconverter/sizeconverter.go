package sizeconverter

import (
	"fmt"
	"math"

	"golang.org/x/exp/constraints"
)

type number interface {
	constraints.Float | constraints.Integer
}

func HumanReadableSizeInMB[N number](size N) string {
	mbs := float64(size) / 1024 / 1024
	// check if mbs is an integer
	if math.Trunc(mbs) == mbs {
		return fmt.Sprintf("%.0f MB", mbs) // round to 0 decimal places
	}
	return fmt.Sprintf("%.2f MB", mbs) // round to 2 decimal places
}
