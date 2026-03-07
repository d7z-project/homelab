package common

import (
	"slices"
	"strings"
)

// SortTags sorts tags according to the system rules:
// - Tags starting with "_" (system tags) come first.
// - Other tags follow.
// - Within each category, tags are sorted alphabetically.
func SortTags(tags []string) {
	if len(tags) <= 1 {
		return
	}
	slices.SortFunc(tags, func(a, b string) int {
		aInt := strings.HasPrefix(a, "_") || strings.HasPrefix(a, "地址池: ") || strings.HasPrefix(a, "策略: ")
		bInt := strings.HasPrefix(b, "_") || strings.HasPrefix(b, "地址池: ") || strings.HasPrefix(b, "策略: ")

		if aInt && !bInt {
			return -1
		}
		if !aInt && bInt {
			return 1
		}
		return strings.Compare(a, b)
	})
}
