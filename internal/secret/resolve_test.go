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

func TestSubstitute_SkipsAsFileRef(t *testing.T) {
	resolved := map[string]string{"creds": "secret-content"}
	out := Substitute("--credentials={$secret.creds.as_file}", resolved)
	// .as_file refs must be left unchanged for SubstituteAsFile pass
	if out != "--credentials={$secret.creds.as_file}" {
		t.Errorf("expected as_file ref unchanged, got %q", out)
	}
}

func TestSubstituteAsFile_Basic(t *testing.T) {
	pathMap := map[string]string{"creds": "/tmp/mcp-secret-abc123"}
	out := SubstituteAsFile("--credentials={$secret.creds.as_file}", pathMap)
	if out != "--credentials=/tmp/mcp-secret-abc123" {
		t.Errorf("got %q, want --credentials=/tmp/mcp-secret-abc123", out)
	}
}

func TestSubstituteAsFile_NotAsFile(t *testing.T) {
	pathMap := map[string]string{"creds": "/tmp/mcp-secret-abc123"}
	// Normal ref should not be touched by SubstituteAsFile
	out := SubstituteAsFile("--token={$secret.creds}", pathMap)
	if out != "--token={$secret.creds}" {
		t.Errorf("expected normal ref unchanged, got %q", out)
	}
}

func TestSubstituteAsFileSlice(t *testing.T) {
	pathMap := map[string]string{"sa": "/tmp/mcp-secret-xyz"}
	ss := []string{"--sa-file={$secret.sa.as_file}", "--plain=foo"}
	out := SubstituteAsFileSlice(ss, pathMap)
	if out[0] != "--sa-file=/tmp/mcp-secret-xyz" {
		t.Errorf("got %q", out[0])
	}
	if out[1] != "--plain=foo" {
		t.Errorf("got %q", out[1])
	}
}

func TestExtractRefsWithModifiers_AsFile(t *testing.T) {
	refs := ExtractRefsWithModifiers("--creds={$secret.sa_key.as_file}")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Name != "sa_key" {
		t.Errorf("name = %q, want sa_key", refs[0].Name)
	}
	if !refs[0].AsFile {
		t.Error("expected AsFile=true")
	}
	if refs[0].Writeback {
		t.Error("expected Writeback=false")
	}
}

func TestExtractRefs_IncludesAsFileRefs(t *testing.T) {
	refs := ExtractRefs("--creds={$secret.sa_key.as_file}")
	if len(refs) != 1 || refs[0] != "sa_key" {
		t.Errorf("expected [sa_key], got %v", refs)
	}
}

func TestSubstitute_WritebackModifierConsumed(t *testing.T) {
	// :w modifier should be stripped; only the resolved value should appear
	resolved := map[string]string{"oauth_creds": "token-value"}
	out := Substitute("{$secret.oauth_creds:w}", resolved)
	if out != "token-value" {
		t.Errorf("got %q, want %q", out, "token-value")
	}
}

func TestSubstitute_WritebackModifierInLargerString(t *testing.T) {
	resolved := map[string]string{"tok": "abc123"}
	out := Substitute("prefix-{$secret.tok:w}-suffix", resolved)
	if out != "prefix-abc123-suffix" {
		t.Errorf("got %q, want %q", out, "prefix-abc123-suffix")
	}
}

func TestExtractRefsWithModifiers_Writeback(t *testing.T) {
	refs := ExtractRefsWithModifiers("{$secret.oauth_token:w}")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Name != "oauth_token" {
		t.Errorf("name = %q, want oauth_token", refs[0].Name)
	}
	if !refs[0].Writeback {
		t.Error("expected Writeback=true")
	}
	if refs[0].AsFile {
		t.Error("expected AsFile=false")
	}
}

func TestExtractRefsWithModifiers_Plain(t *testing.T) {
	refs := ExtractRefsWithModifiers("{$secret.my_token}")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
	if refs[0].Name != "my_token" {
		t.Errorf("name = %q, want my_token", refs[0].Name)
	}
	if refs[0].Writeback {
		t.Error("expected Writeback=false")
	}
	if refs[0].AsFile {
		t.Error("expected AsFile=false")
	}
}

func TestExtractRefsWithModifiers_Mixed(t *testing.T) {
	// string with plain, :w, and .as_file refs
	s := "--a={$secret.plain} --b={$secret.writable:w} --c={$secret.credfile.as_file}"
	refs := ExtractRefsWithModifiers(s)
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d: %+v", len(refs), refs)
	}
	byName := map[string]SecretRef{}
	for _, r := range refs {
		byName[r.Name] = r
	}
	if byName["plain"].Writeback || byName["plain"].AsFile {
		t.Errorf("plain ref should have no modifiers: %+v", byName["plain"])
	}
	if !byName["writable"].Writeback || byName["writable"].AsFile {
		t.Errorf("writable ref should have Writeback=true: %+v", byName["writable"])
	}
	if byName["credfile"].Writeback || !byName["credfile"].AsFile {
		t.Errorf("credfile ref should have AsFile=true: %+v", byName["credfile"])
	}
}
