package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
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

		capabilities, err := readCapabilities(e.Name())
		if err != nil {
			log.Fatal(err)
		}

		connectorCapabilities[e.Name()] = capabilities
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

func readCapabilities(name string) ([]string, error) {
	capabilities := make([]string, 0)
	// Verify the opened file is within the intended directory
	absPath, err := filepath.Abs(*path)
	if err != nil {
		return capabilities, fmt.Errorf("failed to resolve directory %s: %w", *path, err)
	}
	absFile, err := filepath.Abs(filepath.Join(*path, name, "capabilities.go"))
	if err != nil {
		return capabilities, err
	}
	if !strings.HasPrefix(absFile, absPath) {
		return capabilities, fmt.Errorf("invalid path: %s", name)
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filepath.Join(*path, name, "capabilities.go"), nil, 0)
	if err != nil {
		return capabilities, err
	}

	var parseErr error
	ast.Inspect(node, func(n ast.Node) bool {
		if decl, ok := n.(*ast.GenDecl); ok {
			for _, spec := range decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for i, sliceName := range valueSpec.Names {
						if sliceName.Name != "capabilities" {
							continue
						}
						if len(valueSpec.Values) > i {
							if compositeLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
								for _, elt := range compositeLit.Elts {
									str, err := astString(fset, elt)
									if err != nil {
										parseErr = err
										return false
									}
									capabilities = append(capabilities, strings.TrimPrefix(str, "models."))
								}
							}
						}
						return false
					}
				}
			}
		}
		return true
	})
	return capabilities, parseErr
}

// Helper function to convert AST node back to string
func astString(fset *token.FileSet, node ast.Node) (string, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", fmt.Errorf("couldn't covert ast node into string: %w", err)
	}
	return buf.String(), nil
}
