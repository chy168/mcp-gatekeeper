package secret

import (
	"context"
	"regexp"
	"strings"
)

// secretPattern matches {$secret.name} and {$secret.name:modifiers}
var secretPattern = regexp.MustCompile(`\{\$secret\.([a-zA-Z0-9_\-]+)(?::([a-z]+))?\}`)

// SecretRef holds a parsed {$secret.name} or {$secret.name:modifiers} reference.
type SecretRef struct {
	Name      string
	Writeback bool // true when the :w modifier is present
}

// ExtractRefs returns all unique secret key names referenced in s.
// Modifiers (e.g. :w) are stripped — use ExtractRefsWithModifiers to retain them.
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

// ExtractRefsWithModifiers returns all unique secret refs with their modifiers.
func ExtractRefsWithModifiers(s string) []SecretRef {
	matches := secretPattern.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	var result []SecretRef
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			result = append(result, SecretRef{
				Name:      name,
				Writeback: len(m) > 2 && strings.Contains(m[2], "w"),
			})
		}
	}
	return result
}

// resolveAllWithBackend fetches the YAML bundle from backend and looks up names.
func resolveAllWithBackend(backend Backend, bundleName string, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return map[string]string{}, nil
	}

	ctx := context.Background()
	content, err := backend.Get(ctx, bundleName)
	if err != nil {
		return nil, err
	}

	bundle, err := ParseBundle(bundleName, content)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(names))
	for _, name := range names {
		val, err := LookupBundle(bundle, name, bundleName)
		if err != nil {
			return nil, err
		}
		out[name] = val
	}
	return out, nil
}

// ResolveAllWithBackend is like ResolveAll but accepts a pre-created Backend.
// Use this when you need to reuse the backend (e.g. for write-back after subprocess exit).
func ResolveAllWithBackend(backend Backend, bundleName string, names []string) (map[string]string, error) {
	return resolveAllWithBackend(backend, bundleName, names)
}

// ResolveAll fetches the YAML bundle named bundleName from the given source
// backend, parses it, and returns a map of the requested keys → values.
// Fails fast on any error (backend fetch, YAML parse, missing key).
func ResolveAll(source, bundleName string, names []string) (map[string]string, error) {
	backend, err := NewBackend(source)
	if err != nil {
		return nil, err
	}
	return resolveAllWithBackend(backend, bundleName, names)
}

// Substitute replaces all {$secret.name} and {$secret.name:modifiers} occurrences
// in s using the resolved map. Modifiers are consumed and not emitted in the output.
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
