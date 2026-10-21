package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoVersionsFromModule(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantErr bool
	}{
		{"preserves minimum", "module example.com/plugin\n\ngo 1.26.0\n\ntoolchain go1.27.3\n", false},
		{"missing go", "module example.com/plugin\n\ntoolchain go1.27.3\n", true},
		{"missing toolchain", "module example.com/plugin\n\ngo 1.26.0\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			goVersion, toolchain, err := goVersionsFromModule(root)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected missing directive error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if goVersion != "1.26.0" || toolchain != "go1.27.3" {
				t.Fatalf("got go %q, toolchain %q", goVersion, toolchain)
			}
		})
	}
}
