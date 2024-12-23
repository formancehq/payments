package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
)

var (
	connectorDirPath = flag.String("connector-dir-path", "", "Path where to create the new connector directory")
	connectorName    = flag.String("connector-name", "", "Name of the new connector")
)

func main() {
	flag.Parse()

	if *connectorDirPath == "" {
		log.Fatal("connector-dir-path flag is required")
	}

	if *connectorName == "" {
		log.Fatal("connector-name flag is required")
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
