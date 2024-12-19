package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

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
	caser := cases.Title(language.English)

	entries, err := os.ReadDir(*path)
	if err != nil {
		log.Fatal(err)
	}

	output := V3ConnectorConfigYaml{
		Components: Components{
			Schemas: Schemas{
				V3ConnectorConfig: V3ConnectorConfig{},
				V3Configs:         map[string]V3Config{},
			},
		},
	}

	anyOf := []AnyOf{}
	configs := map[string]V3Config{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		configName := "V3" + caser.String(e.Name()) + "Config"

		config, err := readConfig(e.Name())
		if err != nil {
			log.Fatal(err)
		}

		anyOf = append(anyOf, AnyOf{
			Ref: map[string]string{
				"$ref": "#/components/schemas/" + configName,
			},
		})

		configs[configName] = config
	}

	output.Components.Schemas.V3Configs = configs
	output.Components.Schemas.V3ConnectorConfig.AnyOf = anyOf

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

func readConfig(name string) (V3Config, error) {
	f, err := os.Open(*path + "/" + name + "/" + "config.json")
	if err != nil {
		return V3Config{}, err
	}
	defer f.Close()

	var configJson ConfigJson
	if err := json.NewDecoder(f).Decode(&configJson); err != nil {
		return V3Config{}, err
	}

	required := []string{"name"}
	// Add default configs
	var properties = map[string]Property{
		"name": {
			Type: "string",
		},
		"pollingPeriod": {
			Type:    "string",
			Default: "2m",
		},
		"pageSize": {
			Type:    "integer",
			Default: "25",
		},
	}
	for k, v := range configJson {
		if v.Required {
			required = append(required, k)
		}

		properties[k] = Property{
			Type:    v.DataType,
			Default: v.DefaultValue,
		}
	}

	return V3Config{
		Type:       "object",
		Required:   required,
		Properties: properties,
	}, nil
}
