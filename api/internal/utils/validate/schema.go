package validate

import (
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

func (v *JSONValidator) SerializeErrors(detailedError *jsonschema.Detailed, errors string) string {
	if len(detailedError.Error) > 0 {
		errors += fmt.Sprintf("\nError in %s: %s ", detailedError.InstanceLocation, detailedError.Error)
	}
	for _, e := range detailedError.Errors {
		errors = v.SerializeErrors(&e, errors)
	}
	return errors
}

func (v *JSONValidator) Validate(data interface{}) error {

	if err := v.Schema.Validate(data); err != nil {
		detailedErr := err.(*jsonschema.ValidationError).DetailedOutput()

		var errors string
		errors = v.SerializeErrors(&detailedErr, errors)

		return fmt.Errorf("Error validating data: %s", errors)
	}

	return nil
}
