package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// semVer2 is the official SemVer 2.0.0 grammar. The injected value must be a
// release-like version, not an arbitrary string.
var semVer2 = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func TestDefaultPluginVersionIsSemVer(t *testing.T) {
	if !semVer2.MatchString(Version) {
		t.Fatalf("default plugin version %q is not SemVer 2.0.0", Version)
	}
	if got := (Plugin{}).Metadata().Version; got != Version {
		t.Fatalf("metadata version = %q, want %q", got, Version)
	}
}

// TestReleaseLikeSemVerIsInjectableAtLinkTime links a probe with -X and runs
// it. A const Version, or a metadata path that ignores the variable, fails here.
func TestReleaseLikeSemVerIsInjectableAtLinkTime(t *testing.T) {
	const injected = "3.14.1-rc.2+build.7"
	if injected == Version {
		t.Fatalf("probe version %q must differ from the default", injected)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain is unavailable: %v", err)
	}

	probe := filepath.Join(t.TempDir(), "versionprobe")
	if runtime.GOOS == "windows" {
		probe += ".exe"
	}
	build := exec.Command(goTool, "build",
		"-ldflags", "-X github.com/formancehq/payments/plugins/fctl/core.Version="+injected,
		"-o", probe, "testdata/versionprobe/main.go")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build version probe: %v\n%s", err, output)
	}

	output, err := exec.Command(probe).CombinedOutput()
	if err != nil {
		t.Fatalf("run version probe: %v\n%s", err, output)
	}
	if got := strings.TrimSpace(string(output)); got != injected {
		t.Fatalf("linked plugin version = %q, want %q", got, injected)
	}
}

// TestComponentBuildForwardsTheRequestedVersion keeps the guest build wired to
// the same variable the link-time test proves.
func TestComponentBuildForwardsTheRequestedVersion(t *testing.T) {
	script, err := os.ReadFile("../scripts/go-component-build.sh")
	if err != nil {
		t.Fatalf("read component build script: %v", err)
	}
	for _, required := range []string{
		"FCTL_PLUGIN_VERSION",
		"-X github.com/formancehq/payments/plugins/fctl/core.Version=",
	} {
		if !strings.Contains(string(script), required) {
			t.Errorf("component build script does not carry %q", required)
		}
	}
}
