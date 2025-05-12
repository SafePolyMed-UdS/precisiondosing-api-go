package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type JSONValidator struct {
	Schema     *jsonschema.Schema
	SchemaJSON json.RawMessage
}

func NewJSONValidator(schemaFile string) (*JSONValidator, error) {
	schemaBytes, err := os.ReadFile(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	c := jsonschema.NewCompiler()

	if err := c.AddResource(schemaFile, bytes.NewReader(schemaBytes)); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := c.Compile(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("error creating JSONValidator: %w", err)
	}

	return &JSONValidator{
		Schema:     schema,
		SchemaJSON: json.RawMessage(schemaBytes),
	}, nil
}

func (v *JSONValidator) Validate(data interface{}) error {

	if err := v.Schema.Validate(data); err != nil {
		detailedErr := err.(*jsonschema.ValidationError).DetailedOutput()

		var errors string
		errors = v.serializeErrors(&detailedErr, errors)

		return fmt.Errorf("error validating data: %s", errors)
	}

	return nil
}

func (v *JSONValidator) serializeErrors(detailedError *jsonschema.Detailed, errors string) string {
	if len(detailedError.Error) > 0 {
		errors += fmt.Sprintf("\nError in %s: %s ", detailedError.InstanceLocation, detailedError.Error)
	}
	for _, e := range detailedError.Errors {
		errors = v.serializeErrors(&e, errors)
	}
	return errors
}
