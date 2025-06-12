package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/formancehq/payments/internal/connectors/plugins/public"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

var (
	connectorCapabilities map[string][]string

	path           = flag.String("path", "./", "Path to the directory")
	outputFilename = flag.String("output", "connector-capabilities.json", "Name of the output file to write")
)

func main() {
	flag.Parse()
	if *path == "" {
		log.Fatal("path flag is required")
	}
	if *outputFilename == "" {
		log.Fatal("output flag is required")
	}

	entries, err := os.ReadDir(*path)
	if err != nil {
		log.Fatal(err)
	}

	connectorCapabilities = make(map[string][]string)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		capabilities, err := registry.GetCapabilities(e.Name())
		if err != nil {
			log.Fatal(err)
		}

		connectorCapabilities[e.Name()] = toString(capabilities)
	}

	d, err := json.Marshal(&connectorCapabilities)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	f, err := os.Create(*outputFilename)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	defer f.Close()

	_, err = f.Write(d)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func toString(list []models.Capability) []string {
	result := make([]string, len(list))
	for i, item := range list {
		result[i] = fmt.Sprintf("CAPABILITY_%s", item.String())
	}
	return result
}
