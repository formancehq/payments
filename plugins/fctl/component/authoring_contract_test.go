package component

import (
	"os"
	"strings"
	"testing"
)

const authoringFctlSDKRevision = "e9b1395f46f3100b381dbe00f5213de28e6df0e1"

func TestAuthoringToolchainPinsTheFctlComponentVersions(t *testing.T) {
	paths := []string{
		"../../../flake.nix",
		"../../../nix/fctl-component-tools.nix",
		"../scripts/check-component-toolchain.sh",
		"../scripts/go-component-build.sh",
	}
	var contents strings.Builder
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		contents.Write(data)
	}
	for _, required := range []string{
		"fctlSDKRevision = \"" + authoringFctlSDKRevision + "\"",
		"componentTools",
		"binaryen",
		"rust-bin.stable.\"1.91.1\".minimal",
		"version = \"0.4.1\"",
		"rev = \"448f6df8f688cee5d6995e96b1ffc31f9bf00742\"",
		"wasmToolsVersion = \"1.239.0\"",
		"sha256-Kgh17i2vnCAfGaBTNmVafYsG+WJ4OatEdOmrCwrRUs4=",
		"sha256-DoeHrw1+LI3aLhSGMFrUH347nZV2ftykRQ/JUSaERlA=",
		"sha256-g6g1gT7cj8U740ew6c+GT3ZVOENU4wqFf10oHni8MOs=",
		"sha256-P39bkgjlqy8/L7iSA7/hQDOfa3WUF+YV3DlcoGZ73jo=",
		"sha256-9wJSNC/clO8M7E840i1lRWJQT7AdRQX468swmZ4O1rg=",
		"sha256-xIfTYJMVP47timzquEYEb9M8BHsj83NjgD44lbzgd+Y=",
		"componentize-go 0.4.1",
		"wasi-virt 0.2.0",
		"wasm-tools 1.239.0",
		"wasm-opt version 124",
		"wasm-opt -Oz --all-features",
	} {
		if !strings.Contains(contents.String(), required) {
			t.Errorf("component authoring contract does not pin %q", required)
		}
	}
}

// The authoring toolchain is built from Rust sources. Carrying it in the
// default development shell makes every Go CI job compile it and hit
// crates.io, so a transient registry rate limit reds an unrelated build. The
// toolchain must therefore be reachable only as explicit flake packages.
func TestAuthoringToolchainIsNotInTheDefaultDevShell(t *testing.T) {
	flake, err := os.ReadFile("../../../flake.nix")
	if err != nil {
		t.Fatalf("read flake.nix: %v", err)
	}
	source := string(flake)

	shellStart := strings.Index(source, "devShells = forEachSupportedSystem")
	if shellStart < 0 {
		t.Fatalf("flake.nix declares no devShells output")
	}
	outputs, devShells := source[:shellStart], source[shellStart:]

	for _, exposed := range []string{
		"inherit (componentTools) componentize-go wasi-virt wasm-tools;",
		"wasm-opt = pkgs.binaryen;",
	} {
		if !strings.Contains(outputs, exposed) {
			t.Errorf("flake packages do not expose %q", exposed)
		}
	}
	for _, leaked := range []string{"componentTools", "binaryen"} {
		if strings.Contains(devShells, leaked) {
			t.Errorf("default dev shell still carries the authoring toolchain (%q)", leaked)
		}
	}
}

// The component build stays reachable from the repository root, and enters the
// isolated tool environment itself. The root pre-commit gate stops at the
// plugin test suite: the component build is a heavier, separate release gate.
func TestRootGatesKeepTheComponentBuildSeparate(t *testing.T) {
	justfile, err := os.ReadFile("../../../Justfile")
	if err != nil {
		t.Fatalf("read Justfile: %v", err)
	}
	source := string(justfile)

	var preCommit string
	for _, line := range strings.Split(source, "\n") {
		if strings.HasPrefix(line, "pre-commit:") {
			preCommit = line
			break
		}
	}
	if preCommit == "" {
		t.Fatalf("Justfile declares no pre-commit recipe")
	}
	if !strings.Contains(preCommit, "fctl-plugin-test") {
		t.Errorf("pre-commit does not reach the fctl plugin test gate: %q", preCommit)
	}
	if strings.Contains(preCommit, "fctl-component-build") {
		t.Errorf("pre-commit reaches the component build gate: %q", preCommit)
	}

	if !strings.Contains(source, "\nfctl-component-build:\n") {
		t.Fatalf("Justfile declares no fctl-component-build recipe")
	}
	for _, required := range []string{
		"nix shell .#componentize-go .#wasi-virt .#wasm-tools .#wasm-opt --command",
		"just build-component",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("fctl-component-build does not isolate its toolchain: missing %q", required)
		}
	}
}
