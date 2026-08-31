//go:generate go run ./generate

package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dlclark/regexp2"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

type ecmaRegexp regexp2.Regexp

func (re *ecmaRegexp) MatchString(value string) bool {
	matched, err := (*regexp2.Regexp)(re).MatchString(value)
	return err == nil && matched
}
func (re *ecmaRegexp) String() string { return (*regexp2.Regexp)(re).String() }
func compileECMA(value string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(value, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	re.MatchTimeout = 100 * time.Millisecond
	return (*ecmaRegexp)(re), nil
}

type Schemas struct {
	audit      *jsonschema.Schema
	scientific *jsonschema.Schema
	execution  *jsonschema.Schema
	manifest   *jsonschema.Schema
}

func New() (*Schemas, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseRegexpEngine(compileECMA)
	resources := map[string]string{
		"urn:claimbounty:schema:audit-request:1.0.0":     auditRequestSchema,
		"urn:claimbounty:schema:scientific-policy:1.0.0": scientificPolicySchema,
		"urn:claimbounty:schema:execution-policy:1.0.0":  executionPolicySchema,
		"urn:claimbounty:schema:case-manifest:1.0.0":     caseManifestSchema,
	}
	for location, source := range resources {
		var document any
		decoder := json.NewDecoder(bytes.NewBufferString(source))
		decoder.UseNumber()
		if err := decoder.Decode(&document); err != nil {
			return nil, errors.New("validation: embedded schema is invalid")
		}
		if err := compiler.AddResource(location, document); err != nil {
			return nil, errors.New("validation: embedded schema registration failed")
		}
	}
	compile := func(location string) (*jsonschema.Schema, error) {
		schema, err := compiler.Compile(location)
		if err != nil {
			return nil, fmt.Errorf("validation: embedded schema compilation failed: %w", err)
		}
		return schema, nil
	}
	audit, err := compile("urn:claimbounty:schema:audit-request:1.0.0")
	if err != nil {
		return nil, err
	}
	scientific, err := compile("urn:claimbounty:schema:scientific-policy:1.0.0")
	if err != nil {
		return nil, err
	}
	execution, err := compile("urn:claimbounty:schema:execution-policy:1.0.0")
	if err != nil {
		return nil, err
	}
	manifest, err := compile("urn:claimbounty:schema:case-manifest:1.0.0")
	if err != nil {
		return nil, err
	}
	return &Schemas{audit: audit, scientific: scientific, execution: execution, manifest: manifest}, nil
}

func (schemas *Schemas) ValidateAuditRequest(value []byte) error {
	return validate(schemas.audit, value)
}
func (schemas *Schemas) ValidateScientificPolicy(value []byte) error {
	return validate(schemas.scientific, value)
}
func (schemas *Schemas) ValidateExecutionPolicy(value []byte) error {
	return validate(schemas.execution, value)
}
func (schemas *Schemas) ValidateCaseManifest(value []byte) error {
	return validate(schemas.manifest, value)
}

func validate(schema *jsonschema.Schema, value []byte) error {
	if len(value) == 0 || len(value) > 1<<20 {
		return errors.New("validation: JSON document size is invalid")
	}
	var document any
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return errors.New("validation: JSON document is invalid")
	}
	if err := schema.Validate(document); err != nil {
		return errors.New("validation: JSON document does not match its contract")
	}
	return nil
}
