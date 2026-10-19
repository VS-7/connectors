package jsonschema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"connectors/pkg/jsonschema"
)

type ExampleArgs struct {
	Name    string `json:"name" description:"Nome do usuário"`
	Age     int    `json:"age,omitempty" description:"Idade opcional"`
	IsAdmin bool   `json:"is_admin" description:"Flag de administrador"`
}

func TestFromStruct(t *testing.T) {
	schemaRaw, err := jsonschema.FromStruct(ExampleArgs{})
	if err != nil {
		t.Fatalf("erro ao gerar schema a partir de struct: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(schemaRaw, &parsed); err != nil {
		t.Fatalf("schema gerado não é JSON válido: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("esperava type=object, obteve %v", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties não é mapa")
	}

	if _, exists := props["name"]; !exists {
		t.Errorf("esperava propriedade 'name'")
	}
	if _, exists := props["age"]; !exists {
		t.Errorf("esperava propriedade 'age'")
	}
	if _, exists := props["is_admin"]; !exists {
		t.Errorf("esperava propriedade 'is_admin'")
	}

	schemaStr := string(schemaRaw)
	if !strings.Contains(schemaStr, "Nome do usuário") {
		t.Errorf("esperava descrição do campo name no schema")
	}
}
