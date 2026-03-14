package secret

import (
	"reflect"
	"sort"
	"testing"
)

func TestExtractRefs_Basic(t *testing.T) {
	refs := ExtractRefs("hello {$secret.mykey} world")
	if len(refs) != 1 || refs[0] != "mykey" {
		t.Errorf("expected [mykey], got %v", refs)
	}
}

func TestExtractRefs_Multiple(t *testing.T) {
	refs := ExtractRefs("{$secret.foo} and {$secret.bar}")
	sort.Strings(refs)
	want := []string{"bar", "foo"}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("expected %v, got %v", want, refs)
	}
}

func TestExtractRefs_Deduplication(t *testing.T) {
	refs := ExtractRefs("{$secret.foo} and {$secret.foo} again")
	if len(refs) != 1 || refs[0] != "foo" {
		t.Errorf("expected deduped [foo], got %v", refs)
	}
}

func TestExtractRefs_NoRefs(t *testing.T) {
	refs := ExtractRefs("no secrets here")
	if len(refs) != 0 {
		t.Errorf("expected empty slice, got %v", refs)
	}
}

func TestExtractRefs_WithHyphensAndUnderscores(t *testing.T) {
	refs := ExtractRefs("{$secret.my-secret_key}")
	if len(refs) != 1 || refs[0] != "my-secret_key" {
		t.Errorf("expected [my-secret_key], got %v", refs)
	}
}

func TestSubstitute_Basic(t *testing.T) {
	resolved := map[string]string{"mykey": "supersecret"}
	out := Substitute("password={$secret.mykey}", resolved)
	if out != "password=supersecret" {
		t.Errorf("got %q, want %q", out, "password=supersecret")
	}
}

func TestSubstitute_Multiple(t *testing.T) {
	resolved := map[string]string{"foo": "aaa", "bar": "bbb"}
	out := Substitute("{$secret.foo}:{$secret.bar}", resolved)
	if out != "aaa:bbb" {
		t.Errorf("got %q, want %q", out, "aaa:bbb")
	}
}

func TestSubstitute_NoRefs(t *testing.T) {
	resolved := map[string]string{"foo": "aaa"}
	s := "no refs here"
	out := Substitute(s, resolved)
	if out != s {
		t.Errorf("expected unchanged string, got %q", out)
	}
}

func TestSubstitute_UnresolvableRef(t *testing.T) {
	resolved := map[string]string{}
	out := Substitute("{$secret.missing}", resolved)
	// Should remain unchanged when key not in resolved map
	if out != "{$secret.missing}" {
		t.Errorf("expected unchanged ref, got %q", out)
	}
}

func TestExtractRefsFromSlice(t *testing.T) {
	ss := []string{
		"--arg={$secret.foo}",
		"--other={$secret.bar}",
		"--plain=value",
		"--dup={$secret.foo}",
	}
	refs := ExtractRefsFromSlice(ss)
	sort.Strings(refs)
	want := []string{"bar", "foo"}
	if !reflect.DeepEqual(refs, want) {
		t.Errorf("expected %v, got %v", want, refs)
	}
}

func TestSubstituteSlice(t *testing.T) {
	resolved := map[string]string{"key": "value123"}
	ss := []string{"--token={$secret.key}", "--plain=foo"}
	out := SubstituteSlice(ss, resolved)
	if out[0] != "--token=value123" {
		t.Errorf("got %q, want %q", out[0], "--token=value123")
	}
	if out[1] != "--plain=foo" {
		t.Errorf("got %q, want %q", out[1], "--plain=foo")
	}
}
