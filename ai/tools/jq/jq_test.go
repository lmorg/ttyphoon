package jq

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/eino-contrib/jsonschema"
)

func TestJqStructuredInputSchemaAndJSONValue(t *testing.T) {
	inputType := (&Jq{}).InputType()
	if inputType != reflect.TypeOf(jqInputT{}) {
		t.Fatalf("InputType() = %v, want jqInputT", inputType)
	}

	schema := jsonschema.ReflectFromType(inputType)
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	for _, field := range []string{`"query"`, `"json"`} {
		if !strings.Contains(string(schemaJSON), field) {
			t.Fatalf("schema %s does not include %s", schemaJSON, field)
		}
	}

	decoder := json.NewDecoder(strings.NewReader(`{"query":".n","json":{"n":9007199254740993}}`))
	decoder.UseNumber()
	var input jqInputT
	if err := decoder.Decode(&input); err != nil {
		t.Fatalf("decode structured jq input: %v", err)
	}
	jsonValue, ok := input.JSON.(map[string]any)
	if !ok {
		t.Fatalf("JSON value type = %T, want map[string]any", input.JSON)
	}
	if got := jsonValue["n"].(json.Number).String(); got != "9007199254740993" {
		t.Fatalf("JSON number = %s, want exact integer", got)
	}
}
