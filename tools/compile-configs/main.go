package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var (
	path           = flag.String("path", "./", "Path to the directory")
	outputFilename = flag.String("output", "v3-connectors-config.yaml", "Name of the output file to write")
)

func main() {
	flag.Parse()
	if *path == "" {
		log.Fatal("path flag is required")
	}
	if *outputFilename == "" {
		log.Fatal("output flag is required")
	}
	caser := cases.Title(language.English)

	entries, err := os.ReadDir(*path)
	if err != nil {
		log.Fatal(err)
	}

	output := V3ConnectorConfigYaml{
		Components: Components{
			Schemas: Schemas{
				V3ConnectorConfig: V3ConnectorConfig{
					Discriminator: Discriminator{
						PropertyName: "provider",
					},
				},
				V3Configs: map[string]V3Config{},
			},
		},
	}

	oneOf := []OneOf{}
	mapping := map[string]string{}
	configs := map[string]V3Config{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		configName := "V3" + caser.String(e.Name()) + "Config"

		config, err := readConfig(e.Name(), caser.String(e.Name()))
		if err != nil {
			log.Fatal(err)
		}

		mapping[caser.String(e.Name())] = "#/components/schemas/" + configName
		oneOf = append(oneOf, OneOf{
			Ref: map[string]string{
				"$ref": "#/components/schemas/" + configName,
			},
		})

		configs[configName] = config
	}

	output.Components.Schemas.V3Configs = configs
	output.Components.Schemas.V3ConnectorConfig.OneOf = oneOf
	output.Components.Schemas.V3ConnectorConfig.Discriminator.Mapping = mapping

	d, err := yaml.Marshal(&output)
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

func readConfig(name string, caserName string) (V3Config, error) {
	// Verify the opened file is within the intended directory
	absPath, err := filepath.Abs(*path)
	if err != nil {
		return V3Config{}, err
	}
	absFile, err := filepath.Abs(filepath.Join(*path, name, "config.go"))
	if err != nil {
		return V3Config{}, err
	}
	if !strings.HasPrefix(absFile, absPath) {
		return V3Config{}, fmt.Errorf("invalid path: %s", name)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(*path, name, "config.go"), nil, 0)
	if err != nil {
		return V3Config{}, err
	}

	required := []string{"name"}
	var properties = map[string]Property{
		"provider": {
			Type:    "string",
			Default: caserName,
		},
		"name": {
			Type: "string",
		},
		"pollingPeriod": {
			Type:    "string",
			Default: "2m",
		},
		"pageSize": {
			Type:    "integer",
			Default: 25,
		},
	}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}

		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			for _, field := range st.Fields.List {
				if len(field.Names) == 0 ||
					len(field.Names[0].Name) == 0 ||
					unicode.IsLower(rune(field.Names[0].Name[0])) {
					continue
				}

				name := ""
				tagValue := strings.Trim(field.Tag.Value, "`")
				arr := strings.Split(tagValue, " ")
				for _, tag := range arr {
					fields := strings.Split(tag, ":")
					if len(fields) < 2 {
						return V3Config{}, fmt.Errorf("invalid tag: %s", tag)
					}

					switch fields[0] {
					case "json":
						name = strings.Trim(fields[1], "\"")
						typ := field.Type.(*ast.Ident).Name
						fieldType := ""
						switch typ {
						case "string":
							fieldType = "string"
						case "int", "int32", "int64", "uint32", "uint64":
							fieldType = "integer"
						case "bool":
							fieldType = "boolean"
						default:
							return V3Config{}, fmt.Errorf("invalid type: %s", typ)
						}
						properties[name] = Property{
							Type: fieldType,
						}
					case "validate":
						if strings.Contains(fields[1], "required") {
							required = append(required, name)
						}
					}
				}
			}
		}
	}

	return V3Config{
		Type:       "object",
		Required:   required,
		Properties: properties,
	}, nil
}
