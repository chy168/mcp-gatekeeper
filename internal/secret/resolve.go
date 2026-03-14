package secret

import (
	"context"
	"regexp"
	"strings"
)

var secretPattern = regexp.MustCompile(`\{\$secret\.([a-zA-Z0-9_\-]+)\}`)

// ExtractRefs returns all unique secret names referenced in s.
func ExtractRefs(s string) []string {
	matches := secretPattern.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	var result []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}

// resolveAllWithBackend fetches all named secrets using the provided backend.
// Returns a map of name → resolved value. Fails fast on any error.
func resolveAllWithBackend(backend Backend, names []string) (map[string]string, error) {
	ctx := context.Background()
	out := make(map[string]string, len(names))
	for _, name := range names {
		val, err := backend.Get(ctx, name)
		if err != nil {
			return nil, err
		}
		out[name] = val
	}
	return out, nil
}

// ResolveAll fetches all named secrets from the given source backend.
// Returns a map of name → resolved value. Fails fast on any error.
func ResolveAll(source string, names []string) (map[string]string, error) {
	backend, err := NewBackend(source)
	if err != nil {
		return nil, err
	}
	return resolveAllWithBackend(backend, names)
}

// Substitute replaces all {$secret.name} occurrences in s using the resolved map.
func Substitute(s string, resolved map[string]string) string {
	return secretPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := secretPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		name := sub[1]
		if val, ok := resolved[name]; ok {
			return val
		}
		return match
	})
}

// ExtractRefsFromSlice returns all unique secret names referenced across a slice of strings.
func ExtractRefsFromSlice(ss []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, s := range ss {
		for _, name := range ExtractRefs(s) {
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

// SubstituteSlice applies Substitute to each element of a slice.
func SubstituteSlice(ss []string, resolved map[string]string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = Substitute(s, resolved)
	}
	return out
}

// HasRefs reports whether s contains any {$secret.*} references.
func HasRefs(s string) bool {
	return strings.Contains(s, "{$secret.")
}
