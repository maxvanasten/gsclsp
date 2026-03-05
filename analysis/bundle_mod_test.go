package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleModForURIReplacesExistingBundleAndCopiesRecursively(t *testing.T) {
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "zm_tomb_challenge")
	if err := os.MkdirAll(filepath.Join(sourceRoot, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(sourceRoot, ".git", "objects"), 0o755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}

	writeTestFile(t, filepath.Join(sourceRoot, "main.gsc"), "main(){}")
	writeTestFile(t, filepath.Join(sourceRoot, "nested", "util.gsc"), "util(){}")
	writeTestFile(t, filepath.Join(sourceRoot, ".git", "objects", "ignored.gsc"), "ignored(){}")

	bundleRoot := filepath.Join(sourceRoot, "zm_tomb_challenge")
	writeTestFile(t, filepath.Join(bundleRoot, "scripts", "stale.gsc"), "stale(){}")
	writeTestFile(t, filepath.Join(bundleRoot, "old.txt"), "old")

	uri := bundleTestPathToURI(filepath.Join(sourceRoot, "main.gsc"))
	result, err := BundleModForURI(uri)
	if err != nil {
		t.Fatalf("BundleModForURI failed: %v", err)
	}
	if !strings.Contains(result, "Bundled 2 script(s)") {
		t.Fatalf("unexpected result: %q", result)
	}

	if _, err := os.Stat(filepath.Join(bundleRoot, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bundleRoot, "scripts", "stale.gsc")); !os.IsNotExist(err) {
		t.Fatalf("expected stale script to be removed, stat err: %v", err)
	}

	assertFileExists(t, filepath.Join(bundleRoot, "scripts", "main.gsc"))
	assertFileExists(t, filepath.Join(bundleRoot, "scripts", "nested", "util.gsc"))
	if _, err := os.Stat(filepath.Join(bundleRoot, "scripts", ".git", "objects", "ignored.gsc")); !os.IsNotExist(err) {
		t.Fatalf("expected hidden directory file to be skipped, stat err: %v", err)
	}

	assertFileExists(t, filepath.Join(sourceRoot, "main.gsc"))
	assertFileExists(t, filepath.Join(sourceRoot, "nested", "util.gsc"))

	manifestPath := filepath.Join(bundleRoot, "mod.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read mod.json: %v", err)
	}
	var manifest map[string]string
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("unmarshal mod.json: %v", err)
	}
	if manifest["name"] != "zm_tomb_challenge" {
		t.Fatalf("unexpected mod name: %q", manifest["name"])
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}

func bundleTestPathToURI(path string) string {
	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, "/") {
		return "file://" + path
	}
	return "file:///" + path
}
