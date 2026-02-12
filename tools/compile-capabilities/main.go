package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/formancehq/payments/ee/plugins"
	_ "github.com/formancehq/payments/internal/connectors/plugins/public"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

type pathList []string

func (p *pathList) String() string { return strings.Join(*p, ",") }
func (p *pathList) Set(value string) error {
	*p = append(*p, value)
	return nil
}

var (
	connectorCapabilities map[string][]string

	paths          pathList
	outputFilename = flag.String("output", "connector-capabilities.json", "Name of the output file to write")
)

func main() {
	flag.Var(&paths, "path", "Path to a plugins directory (can be specified multiple times)")
	flag.Parse()
	if len(paths) == 0 {
		log.Fatal("at least one --path flag is required")
	}
	if *outputFilename == "" {
		log.Fatal("output flag is required")
	}

	connectorCapabilities = make(map[string][]string)
	for _, p := range paths {
		entries, err := os.ReadDir(p)
		if err != nil {
			log.Fatal(err)
		}

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
