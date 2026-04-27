package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/grovetools/grove-gemini/pkg/config"
	"github.com/invopop/jsonschema"
)

func main() {
	r := &jsonschema.Reflector{
		AllowAdditionalProperties: true,
		ExpandedStruct:            true,
		FieldNameTag:              "yaml",
	}

	schema := r.Reflect(&config.GeminiConfig{})
	schema.Title = "Grove Gemini Configuration"
	schema.Description = "Schema for the 'gemini' extension in grove.yml."

	// Make all fields optional - Grove configs should not require any fields
	schema.Required = nil

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling schema: %v", err)
	}

	// Write to the package root
	if err := os.WriteFile("gemini.schema.json", data, 0o644); err != nil { //nolint:gosec // schema file needs to be readable
		log.Fatalf("Error writing schema file: %v", err)
	}

	log.Printf("Successfully generated gemini schema at gemini.schema.json")
}
