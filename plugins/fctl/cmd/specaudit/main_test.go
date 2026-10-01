package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specPath is the merged document, relative to this package directory.
const specPath = "../../../../openapi.yaml"

// moduleRoot is the plugin module root, relative to this package directory.
const moduleRoot = "../.."

// TestCheckPassesAgainstCommittedArtefacts is the gate a pre-commit run needs:
// -check must succeed against what is committed, without writing anything.
func TestCheckPassesAgainstCommittedArtefacts(t *testing.T) {
	before := snapshot(t)

	if err := run(specPath, moduleRoot, true); err != nil {
		t.Fatalf("check against committed artefacts failed: %v", err)
	}

	if after := snapshot(t); after != before {
		t.Error("check mode modified a committed artefact")
	}
}

// TestCheckFailsOnDrift asserts the gate actually detects drift rather than
// passing unconditionally.
func TestCheckFailsOnDrift(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "audit", "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		filepath.Join("audit", "testdata", "report.json"),
		filepath.Join("docs", "v3-operations.generated.md"),
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte("stale\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := run(specPath, dir, true); err == nil {
		t.Error("check mode accepted stale artefacts")
	}
}

// TestWriteIsReproducible asserts a fresh generation into an empty tree matches
// the committed artefacts byte for byte.
func TestWriteIsReproducible(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "audit", "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := run(specPath, dir, false); err != nil {
		t.Fatalf("generate: %v", err)
	}

	for _, rel := range []string{
		filepath.Join("audit", "testdata", "report.json"),
		filepath.Join("docs", "v3-operations.generated.md"),
	} {
		got, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(moduleRoot, rel))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from the committed artefact", rel)
		}
	}
}

func TestCommandCheckPassesWithExplicitPaths(t *testing.T) {
	var stderr bytes.Buffer
	code := command([]string{"-spec", specPath, "-out", moduleRoot, "-check"}, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("command exit=%d stderr=%q", code, stderr.String())
	}
}

func TestCommandReportsFlagAndAuditFailures(t *testing.T) {
	t.Run("unknown flag", func(t *testing.T) {
		var stderr bytes.Buffer
		if code := command([]string{"-unknown"}, &stderr); code != 2 {
			t.Fatalf("exit=%d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "flag provided but not defined") {
			t.Fatalf("stderr=%q", stderr.String())
		}
	})

	t.Run("invalid spec", func(t *testing.T) {
		var stderr bytes.Buffer
		if code := command([]string{"-spec", filepath.Join(t.TempDir(), "missing.yaml"), "-check"}, &stderr); code != 1 {
			t.Fatalf("exit=%d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "specaudit:") {
			t.Fatalf("stderr=%q", stderr.String())
		}
	})
}

func snapshot(t *testing.T) string {
	t.Helper()
	var out string
	for _, rel := range []string{
		filepath.Join("audit", "testdata", "report.json"),
		filepath.Join("docs", "v3-operations.generated.md"),
	} {
		data, err := os.ReadFile(filepath.Join(moduleRoot, rel))
		if err != nil {
			t.Fatal(err)
		}
		out += rel + "\x00" + string(data) + "\x00"
	}
	return out
}
