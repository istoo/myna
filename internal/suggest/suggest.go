// Package suggest finds the names most like what a user typed, for a
// "did you mean" line.
package suggest

import (
	"slices"
	"strings"
)

// Close returns the names most like in, closest first, at most n: those
// containing it, then those a few edits away. A name with a "/" is
// compared on the part after it, since the part before may be the one
// that is wrong (a model id's provider).
func Close(in string, names []string, n int) []string {
	q := strings.ToLower(in)
	if _, m, ok := strings.Cut(q, "/"); ok && m != "" {
		q = m
	}
	type hit struct {
		id string
		d  int
	}
	var hits []hit
	for _, id := range names {
		l := strings.ToLower(id)
		d := editDistance(q, l[strings.Index(l, "/")+1:])
		switch {
		case strings.Contains(l, q):
			d = -1
		case d > max(2, len(q)/3):
			continue
		}
		hits = append(hits, hit{id, d})
	}
	slices.SortStableFunc(hits, func(a, b hit) int { return a.d - b.d })
	var out []string
	for _, h := range hits {
		if len(out) == n {
			break
		}
		out = append(out, h.id)
	}
	return out
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			c := 1
			if a[i-1] == b[j-1] {
				c = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+c)
		}
		prev = cur
	}
	return prev[len(b)]
}
