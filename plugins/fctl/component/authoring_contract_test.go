package component

import (
	"os"
	"strings"
	"testing"
)

const authoringFctlSDKRevision = "545521bfa222250af6b4419b194c7967cded0379"

func TestAuthoringDevShellPinsTheFctlComponentToolchain(t *testing.T) {
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
		"componentTools.componentize-go",
		"componentTools.wasi-virt",
		"componentTools.wasm-tools",
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
