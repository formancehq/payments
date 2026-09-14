package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	workflowPath = "../../../.github/workflows/main.yml"
	justfilePath = "../../../Justfile"
	wrapperPath  = "with-fctl-sdk.sh"
)

// workflow is the subset of the repository's default pipeline this contract
// constrains: the delegated jobs that run `just tests` and `just pre-commit`.
type workflow struct {
	Jobs map[string]struct {
		Uses    string            `yaml:"uses"`
		Secrets map[string]string `yaml:"secrets"`
	} `yaml:"jobs"`
}

func readWorkflow(t *testing.T) workflow {
	t.Helper()
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}
	var parsed workflow
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse %s: %v", workflowPath, err)
	}
	return parsed
}

// TestDefaultPipelineGrantsTheFctlSDKCredentialToEveryJobRunningThePluginGate
// pins the only mechanism by which ordinary CI can reach the pinned fctl SDK:
// formancehq/ci's setup-nix installs a github.com/formancehq/ credential from
// GIT_PRIVATE_TOKEN, and the SDK repository is private.
func TestDefaultPipelineGrantsTheFctlSDKCredentialToEveryJobRunningThePluginGate(t *testing.T) {
	parsed := readWorkflow(t)
	for _, name := range []string{"Tests", "Dirty"} {
		job, ok := parsed.Jobs[name]
		if !ok {
			t.Errorf("workflow declares no %s job", name)
			continue
		}
		if _, ok := job.Secrets["GIT_PRIVATE_TOKEN"]; !ok {
			t.Errorf("%s job does not receive GIT_PRIVATE_TOKEN, so it cannot materialise the pinned fctl SDK", name)
		}
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

// TestSDKWrapperMaterialisesTheLockedRevisionWithoutAFloatingReference proves
// the wrapper can obtain the SDK on a machine with no prior checkout, and that
// it never resolves a moving reference to do so.
func TestSDKWrapperMaterialisesTheLockedRevisionWithoutAFloatingReference(t *testing.T) {
	data, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read %s: %v", wrapperPath, err)
	}
	wrapper := string(data)
	for _, required := range []string{
		"FCTL_SDK_ROOT",                 // the local-development override survives
		"FCTL_SDK_CACHE_DIR",            // the materialised cache is relocatable
		"$expected_commit",              // materialisation is pinned to the locked revision
		"git -C \"$staging_directory\"", // materialisation happens in a throwaway repository
	} {
		if !strings.Contains(wrapper, required) {
			t.Errorf("wrapper does not carry %q", required)
		}
	}
	floating := regexp.MustCompile(`(?m)^[^#\n]*\bgit\b[^\n]*\bfetch\b[^\n]*\b(HEAD|main|master|refs/heads/[a-zA-Z0-9_/-]+|--tags)\b`)
	if match := floating.FindString(wrapper); match != "" {
		t.Errorf("wrapper fetches a floating reference: %s", strings.TrimSpace(match))
	}
}
