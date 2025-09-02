package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

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
		map[string]interface{}{
			"Connector": *connectorName,
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
