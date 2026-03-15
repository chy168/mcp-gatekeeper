package proxy

import (
	"os"
	"strings"
	"testing"
)

func TestApplyFileInjections_TempFile(t *testing.T) {
	content := "super-secret-value"
	envs, tmpFiles, _, err := applyFileInjections([]string{"MY_CRED=" + content}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 env addition, got %d", len(envs))
	}
	if len(tmpFiles) != 1 {
		t.Fatalf("expected 1 temp file, got %d", len(tmpFiles))
	}

	// Check env entry is VAR=path
	parts := strings.SplitN(envs[0], "=", 2)
	if parts[0] != "MY_CRED" {
		t.Errorf("expected env key MY_CRED, got %q", parts[0])
	}
	tmpPath := parts[1]

	// Check file contents
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("could not read temp file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}

	// Check permissions
	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("stat temp file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file perm = %o, want 0600", info.Mode().Perm())
	}

	// Cleanup
	os.Remove(tmpPath)
}

func TestApplyFileInjections_FixedPath(t *testing.T) {
	tmpDir := t.TempDir()
	fixedPath := tmpDir + "/creds.json"
	content := `{"key":"value"}`

	envs, tmpFiles, _, err := applyFileInjections([]string{fixedPath + "=" + content}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(envs) != 0 {
		t.Errorf("fixed path should produce no env additions, got %v", envs)
	}
	if len(tmpFiles) != 0 {
		t.Errorf("fixed path should produce no temp files, got %v", tmpFiles)
	}

	data, err := os.ReadFile(fixedPath)
	if err != nil {
		t.Fatalf("could not read fixed path file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}

	info, err := os.Stat(fixedPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestApplyFileInjections_InvalidEntry(t *testing.T) {
	_, _, _, err := applyFileInjections([]string{"NO_EQUALS_SIGN"}, nil)
	if err == nil {
		t.Fatal("expected error for missing '=', got nil")
	}
}

func TestApplyFileInjections_TempFileCleanup(t *testing.T) {
	_, tmpFiles, _, err := applyFileInjections([]string{"FOO=bar"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tmpFiles) != 1 {
		t.Fatalf("expected 1 temp file, got %d", len(tmpFiles))
	}
	// Simulate cleanup
	for _, f := range tmpFiles {
		os.Remove(f)
	}
	// Verify deleted
	if _, err := os.Stat(tmpFiles[0]); !os.IsNotExist(err) {
		t.Errorf("temp file still exists after cleanup: %s", tmpFiles[0])
	}
}

func TestApplyFileInjections_WithSecretSubstitution(t *testing.T) {
	resolved := map[string]string{"my_token": "resolved-value"}
	envs, tmpFiles, _, err := applyFileInjections([]string{"TOKEN={$secret.my_token}"}, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	if len(envs) != 1 || len(tmpFiles) != 1 {
		t.Fatalf("expected 1 env + 1 temp file, got %d + %d", len(envs), len(tmpFiles))
	}
	data, _ := os.ReadFile(tmpFiles[0])
	if string(data) != "resolved-value" {
		t.Errorf("file content = %q, want %q", string(data), "resolved-value")
	}
}

func TestApplyFileInjections_Writeback(t *testing.T) {
	resolved := map[string]string{"oauth_token": "initial-value"}
	envs, tmpFiles, writebacks, err := applyFileInjections([]string{"TOKEN={$secret.oauth_token:w}"}, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	if len(envs) != 1 || len(tmpFiles) != 1 {
		t.Fatalf("expected 1 env + 1 temp file, got %d + %d", len(envs), len(tmpFiles))
	}
	if len(writebacks) != 1 {
		t.Fatalf("expected 1 writeback, got %d", len(writebacks))
	}
	if writebacks[0].SecretKey != "oauth_token" {
		t.Errorf("writeback key = %q, want %q", writebacks[0].SecretKey, "oauth_token")
	}
	if writebacks[0].Path != tmpFiles[0] {
		t.Errorf("writeback path should match temp file path")
	}
}

func TestApplyFileInjections_NoWritebackWithoutModifier(t *testing.T) {
	resolved := map[string]string{"my_token": "value"}
	_, tmpFiles, writebacks, err := applyFileInjections([]string{"TOKEN={$secret.my_token}"}, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	if len(writebacks) != 0 {
		t.Errorf("expected no writebacks without :w modifier, got %d", len(writebacks))
	}
}
