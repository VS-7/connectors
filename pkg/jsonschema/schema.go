package jsonschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Property descreve uma propriedade em um JSON Schema de objeto.
type Property struct {
	Type        string               `json:"type"`
	Description string               `json:"description,omitempty"`
	Enum        []string             `json:"enum,omitempty"`
	Items       *Property            `json:"items,omitempty"`
	Properties  map[string]*Property `json:"properties,omitempty"`
	Required    []string             `json:"required,omitempty"`
}

// ObjectSchema constrói um JSON Schema Draft 7 para um objeto com propriedades tipadas.
type ObjectSchema struct {
	Type       string               `json:"type"`
	Properties map[string]*Property `json:"properties"`
	Required   []string             `json:"required,omitempty"`
}

// NewObjectSchema inicializa um builder de esquema para objetos.
func NewObjectSchema() *ObjectSchema {
	return &ObjectSchema{
		Type:       "object",
		Properties: make(map[string]*Property),
		Required:   make([]string, 0),
	}
}

// AddProperty adiciona uma propriedade tipada ao esquema.
func (s *ObjectSchema) AddProperty(name, propType, description string, required bool) *ObjectSchema {
	s.Properties[name] = &Property{
		Type:        propType,
		Description: description,
	}
	if required {
		s.Required = append(s.Required, name)
	}
	return s
}

// AddEnumProperty adiciona uma propriedade com valores permitidos (enum).
func (s *ObjectSchema) AddEnumProperty(name, description string, values []string, required bool) *ObjectSchema {
	s.Properties[name] = &Property{
		Type:        "string",
		Description: description,
		Enum:        values,
	}
	if required {
		s.Required = append(s.Required, name)
	}
	return s
}

// AddArrayProperty adiciona uma propriedade do tipo array com itens de tipo homogêneo.
func (s *ObjectSchema) AddArrayProperty(name, itemType, description string, required bool) *ObjectSchema {
	s.Properties[name] = &Property{
		Type:        "array",
		Description: description,
		Items: &Property{
			Type: itemType,
		},
	}
	if required {
		s.Required = append(s.Required, name)
	}
	return s
}

// RawMessage serializa o esquema em json.RawMessage formatado.
func (s *ObjectSchema) RawMessage() json.RawMessage {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return json.RawMessage(`{"type": "object"}`)
	}
	return json.RawMessage(bytes)
}

// FromStruct reflete sobre uma struct Go e gera automaticamente o JSON Schema correspondente.
// Lê tags `json:"nome,omitempty"` e `jsonschema:"description=..."` ou `description:"..."`.
func FromStruct(v any) (json.RawMessage, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("FromStruct requer uma struct, recebido: %s", t.Kind())
	}

	schema := NewObjectSchema()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		parts := strings.Split(jsonTag, ",")
		fieldName := parts[0]
		if fieldName == "" {
			fieldName = field.Name
		}

		isOptional := false
		for _, p := range parts[1:] {
			if p == "omitempty" {
				isOptional = true
				break
			}
		}

		desc := field.Tag.Get("description")
		if desc == "" {
			desc = field.Tag.Get("jsonschema")
		}

		propType := goTypeToJSONType(field.Type)
		schema.AddProperty(fieldName, propType, desc, !isOptional)
	}

	return schema.RawMessage(), nil
}

func goTypeToJSONType(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}
