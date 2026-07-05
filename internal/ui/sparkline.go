package ui

import (
	"math"
)

var blockChars = []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func GenerateSparkline(data []float64, width int) string {
	if len(data) == 0 {
		return ""
	}

	if len(data) > width {
		data = data[len(data)-width:]
	}

	min := math.MaxFloat64
	max := -math.MaxFloat64
	for _, v := range data {
		if v < 0 {
			continue // Skip timeouts for scaling
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Handle all same values or all timeouts
	if min == max || min == math.MaxFloat64 {
		res := ""
		for _, v := range data {
			if v < 0 {
				res += " "
			} else if max == 0 {
				res += string(blockChars[0])
			} else {
				res += string(blockChars[4])
			}
		}
		return res
	}

	res := ""
	for _, v := range data {
		if v < 0 {
			res += " "
			continue
		}
		normalized := (v - min) / (max - min)
		idx := int(math.Round(normalized * float64(len(blockChars)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blockChars) {
			idx = len(blockChars) - 1
		}
		res += string(blockChars[idx])
	}

	return res
}
