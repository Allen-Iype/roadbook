package timeline

import (
	"encoding/json"
	"sort"
)

// PathCount is one observed JSON key path and how often it occurred.
type PathCount struct {
	Path  string
	Count int
}

// Probe reports every key path present in a Timeline export and its frequency,
// sorted by path. It exists permanently because Google changes the export schema
// without announcement: run it against a new export and diff against a previous
// run to see what moved. Array indices collapse to "[]" so repetition aggregates.
func Probe(data []byte) ([]PathCount, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	counts := map[string]int{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, child := range x {
				p := k
				if prefix != "" {
					p = prefix + "." + k
				}
				counts[p]++
				walk(p, child)
			}
		case []any:
			for _, child := range x {
				walk(prefix+"[]", child)
			}
		}
	}
	walk("", root)
	out := make([]PathCount, 0, len(counts))
	for p, n := range counts {
		out = append(out, PathCount{Path: p, Count: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
