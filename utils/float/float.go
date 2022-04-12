package float

import "math"

func ToTwoPoints32[T float32 | float64](num T) float32 {
	num64 := float64(num)
	return float32(math.Floor(num64*100)) / 100
}

func ToTwoPoints64[T float32 | float64](num T) float64 {
	num64 := float64(num)
	return math.Floor(num64*100) / 100
}
