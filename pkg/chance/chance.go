package chance

import "math/rand"

var Default = New()

type Chancer struct {
}

func New() *Chancer {
	return &Chancer{}
}

func (c *Chancer) Win(percent float64) bool {
	return rand.Float64() < percent
}

type WinInput struct {
	Percent float64
	Action  func()
}

func (c *Chancer) PickWin(inputs ...WinInput) {
	var cumulativeChance float64
	chances := make([]float64, len(inputs))
	for i, item := range inputs {
		cumulativeChance += item.Percent
		chances[i] = cumulativeChance
	}

	r := rand.Float64()
	for i, chance := range chances {
		if r < chance {
			inputs[i].Action()
			return
		}
	}
	// for some reason we didn't pick any input, so pick random one
	inputs[rand.Intn(len(inputs))].Action()
}
