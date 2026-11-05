package spec

import "fmt"

const maxSuggestDistance = 2

func suggestion(got string, known []string) string {
	best, bestDistance := "", maxSuggestDistance+1
	for _, k := range known {
		if d := distance(got, k); d < bestDistance {
			best, bestDistance = k, d
		}
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf(" (did you mean %q?)", best)
}

func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
