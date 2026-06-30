package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const repoModule = "github.com/formancehq/payments"

var (
	connectorDirPath = flag.String("connector-dir-path", "", "Path where to create the new connector directory")
	connectorName    = flag.String("connector-name", "", "Name of the new connector")

	packageNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

func main() {
	flag.Parse()

	if *connectorDirPath == "" {
		log.Fatal("connector-dir-path flag is required")
	}

	if *connectorName == "" {
		log.Fatal("connector-name flag is required")
	}

	if !isValidPackageName(*connectorName) {
		log.Fatalf("connector-name %q contains invalid characters: must be a valid package name", *connectorName)
	}

	connectorPath := filepath.Join(*connectorDirPath, *connectorName)

	absDir, err := filepath.Abs(*connectorDirPath)
	if err != nil {
		log.Fatal(err)
	}
	// connector-dir-path is always <repo-root>/{ce,ee}/plugins, so repo root is two levels up.
	repoRoot := filepath.Dir(filepath.Dir(absDir))

	pluginAbs := filepath.Join(absDir, *connectorName)

	relFromRoot, err := filepath.Rel(repoRoot, pluginAbs)
	if err != nil {
		log.Fatal(err)
	}
	relSlash := filepath.ToSlash(relFromRoot)
	modulePath := repoModule + "/" + relSlash

	// EE plugins live inside the root module; only CE plugins get their own go.mod.
	isCE := strings.HasPrefix(relSlash, "ce/plugins/")

	// Create the new connector's directory
	if err := os.Mkdir(connectorPath, 0755); err != nil {
		log.Fatal(err)
	}

	// Create the new connector client's directory
	if err := os.Mkdir(filepath.Join(connectorPath, "client"), 0755); err != nil {
		log.Fatal(err)
	}

	// Create the new connector's files
	if err := createFiles(
		context.Background(),
		connectorPath,
		isCE,
		map[string]interface{}{
			"Connector": *connectorName,
			"Module":    modulePath,
		},
	); err != nil {
		log.Fatal(err)
	}
}

func isValidPackageName(name string) bool {
	if name == "go" || name == "main" || name == "internal" {
		return false
	}
	return packageNameRegex.MatchString(name)
}
