package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const (
	workflowPath = "../../../.github/workflows/main.yml"
	justfilePath = "../../../Justfile"
	wrapperPath  = "with-fctl-sdk.sh"
)

// TestDefaultPipelineNeedsNoCrossRepositoryCredential pins the property the
// committed SDK snapshot buys: the jobs that run the plugin gate resolve the
// fctl SDK from this tree, so no workflow may hand them a Git credential for
// the private SDK repository. A pull_request-triggered job must not carry one.
func TestDefaultPipelineNeedsNoCrossRepositoryCredential(t *testing.T) {
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}
	if strings.Contains(string(data), "GIT_PRIVATE_TOKEN") {
		t.Error("workflow still passes GIT_PRIVATE_TOKEN; the committed SDK snapshot makes it unnecessary")
	}
}

// TestRepositoryGatesStillRunThePluginSuite keeps the plugin inside the
// ordinary gates rather than behind an opt-in recipe nobody runs.
func TestRepositoryGatesStillRunThePluginSuite(t *testing.T) {
	data, err := os.ReadFile(justfilePath)
	if err != nil {
		t.Fatalf("read %s: %v", justfilePath, err)
	}
	text := string(data)
	preCommit := recipeBody(t, text, "pre-commit")
	for _, required := range []string{"fctl-audit-check", "fctl-audit-tidy-check", "fctl-plugin-test"} {
		if !strings.Contains(preCommit, required) {
			t.Errorf("pre-commit no longer runs %s", required)
		}
	}
	if !strings.Contains(recipeBody(t, text, "tests"), "fctl-plugin-test") {
		t.Error("tests no longer runs fctl-plugin-test")
	}
}

func recipeBody(t *testing.T, justfile, name string) string {
	t.Helper()
	lines := strings.Split(justfile, "\n")
	for index, line := range lines {
		if !strings.HasPrefix(line, name+":") && !strings.HasPrefix(line, name+" ") {
			continue
		}
		body := []string{line}
		for _, next := range lines[index+1:] {
			if next != "" && !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
				break
			}
			body = append(body, next)
		}
		return strings.Join(body, "\n")
	}
	t.Fatalf("Justfile declares no %q recipe", name)
	return ""
}

// TestSDKWrapperResolvesTheCommittedSnapshotWithoutFetching proves the wrapper
// can obtain the SDK on a machine with no prior checkout and no credential, and
// that it reaches no network to do so.
func TestSDKWrapperResolvesTheCommittedSnapshotWithoutFetching(t *testing.T) {
	data, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read %s: %v", wrapperPath, err)
	}
	wrapper := string(data)
	for _, required := range []string{
		"FCTL_SDK_ROOT",             // the local-development override survives
		"$bundle_path",              // the credential-free source is the committed snapshot
		"$expected_bundle_nar_hash", // the snapshot is held to its own content hash
		"$expected_sdk_nar_hash",    // an override is still held to the upstream module hash
	} {
		if !strings.Contains(wrapper, required) {
			t.Errorf("wrapper does not carry %q", required)
		}
	}
	fetching := regexp.MustCompile(`(?m)^[^#\n]*\b(git[^\n]*\b(fetch|clone|ls-remote)|curl|wget)\b`)
	if match := fetching.FindString(wrapper); match != "" {
		t.Errorf("wrapper reaches the network: %s", strings.TrimSpace(match))
	}
}
