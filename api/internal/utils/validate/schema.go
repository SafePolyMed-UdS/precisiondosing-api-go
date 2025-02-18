package validate

import (
	"encoding/json"
	"fmt"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type JSONValidator struct {
	Schema *jsonschema.Schema
}

func NewJSONValidator(schemaFile string) (*JSONValidator, error) {
	c := jsonschema.NewCompiler()
	schema, err := c.Compile(schemaFile)
	if err != nil {
		fmt.Println("Error compiling schema:", err)
		return nil, err
	}

	return &JSONValidator{Schema: schema}, nil
}

func (v *JSONValidator) Validate(data interface{}) error {
	jsonData, err := json.Marshal(data) // Convert struct to JSON
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	if err := v.Schema.Validate(jsonData); err != nil {
		fmt.Println("Error validating data:", err)
		return err
	}

	return nil
}
