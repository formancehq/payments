package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

const cePluginsPrefix = "github.com/formancehq/payments/ce/plugins/"

func main() {
	connectorDirPath := flag.String("connector-dir-path", "", "Path to the ce/plugins directory")
	flag.Parse()

	if *connectorDirPath == "" {
		log.Fatal("connector-dir-path flag is required")
	}

	pluginsDir, err := filepath.Abs(*connectorDirPath)
	if err != nil {
		log.Fatal(err)
	}
	// connector-dir-path is always <repo-root>/ce/plugins, so repo root is two levels up.
	repoRoot := filepath.Dir(filepath.Dir(pluginsDir))

	rootModPath := filepath.Join(repoRoot, "go.mod")
	rootModData, err := os.ReadFile(rootModPath)
	if err != nil {
		log.Fatalf("reading root go.mod: %v", err)
	}
	rootMod, err := modfile.Parse(rootModPath, rootModData, nil)
	if err != nil {
		log.Fatalf("parsing root go.mod: %v", err)
	}

	// Collect live plugins: module path -> relative dir (e.g. "ce/plugins/adyen")
	live := map[string]string{}
	entries, err := os.ReadDir(pluginsDir)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("reading ce/plugins: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		modPath := filepath.Join(pluginsDir, e.Name(), "go.mod")
		data, err := os.ReadFile(modPath)
		if err != nil {
			continue // no go.mod yet, skip
		}
		f, err := modfile.Parse(modPath, data, nil)
		if err != nil {
			log.Printf("warning: parsing %s: %v", modPath, err)
			continue
		}
		if strings.HasPrefix(f.Module.Mod.Path, cePluginsPrefix) {
			rel, err := filepath.Rel(repoRoot, filepath.Join(pluginsDir, e.Name()))
			if err != nil {
				log.Fatalf("computing relative path: %v", err)
			}
			live[f.Module.Mod.Path] = filepath.ToSlash(rel)
		}
	}

	changed := false

	// Drop stale replace/require entries for ce/plugins that no longer exist.
	// Iterate over copies since we modify the slices.
	for _, r := range append([]*modfile.Replace(nil), rootMod.Replace...) {
		if !strings.HasPrefix(r.Old.Path, cePluginsPrefix) {
			continue
		}
		if _, ok := live[r.Old.Path]; !ok {
			fmt.Printf("dropping stale replace: %s\n", r.Old.Path)
			if err := rootMod.DropReplace(r.Old.Path, r.Old.Version); err != nil {
				log.Fatalf("drop replace %s: %v", r.Old.Path, err)
			}
			changed = true
		}
	}
	for _, r := range append([]*modfile.Require(nil), rootMod.Require...) {
		if !strings.HasPrefix(r.Mod.Path, cePluginsPrefix) {
			continue
		}
		if _, ok := live[r.Mod.Path]; !ok {
			fmt.Printf("dropping stale require: %s\n", r.Mod.Path)
			if err := rootMod.DropRequire(r.Mod.Path); err != nil {
				log.Fatalf("drop require %s: %v", r.Mod.Path, err)
			}
			changed = true
		}
	}

	// Index existing directives for O(1) lookup.
	existingReplaces := map[string]bool{}
	for _, r := range rootMod.Replace {
		existingReplaces[r.Old.Path] = true
	}
	existingRequires := map[string]bool{}
	for _, r := range rootMod.Require {
		existingRequires[r.Mod.Path] = true
	}

	// Add missing replace and require entries.
	for modPath, relDir := range live {
		if !existingReplaces[modPath] {
			fmt.Printf("adding replace: %s => ./%s\n", modPath, relDir)
			if err := rootMod.AddReplace(modPath, "", "./"+relDir, ""); err != nil {
				log.Fatalf("add replace %s: %v", modPath, err)
			}
			changed = true
		}
		if !existingRequires[modPath] {
			fmt.Printf("adding require: %s\n", modPath)
			if err := rootMod.AddRequire(modPath, "v0.0.0-00010101000000-000000000000"); err != nil {
				log.Fatalf("add require %s: %v", modPath, err)
			}
			changed = true
		}
	}

	if !changed {
		return
	}

	rootMod.Cleanup()
	out, err := rootMod.Format()
	if err != nil {
		log.Fatalf("formatting go.mod: %v", err)
	}
	if err := os.WriteFile(rootModPath, out, 0644); err != nil {
		log.Fatalf("writing go.mod: %v", err)
	}
	fmt.Println("go.mod updated")
}
