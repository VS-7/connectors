package core_test

import (
	"context"
	"encoding/json"
	"testing"

	"connectors/internal/core"
)

type dummyConnector struct {
	namespace string
	initErr   error
}

func (d *dummyConnector) Namespace() string { return d.namespace }
func (d *dummyConnector) Initialize(ctx context.Context) error { return d.initErr }
func (d *dummyConnector) Tools() []core.ToolDefinition {
	return []core.ToolDefinition{
		{
			Name:        d.namespace + "_ping",
			Description: "Ping da tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
			Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
				return map[string]string{"pong": d.namespace}, nil
			},
		},
	}
}
func (d *dummyConnector) Close() error { return nil }

func TestRegistry(t *testing.T) {
	r := core.NewRegistry()
	c1 := &dummyConnector{namespace: "conn1"}
	c2 := &dummyConnector{namespace: "conn2"}

	r.Register(c1)
	r.Register(c2)

	if len(r.All()) != 2 {
		t.Fatalf("esperava 2 conectores, obteve %d", len(r.All()))
	}

	filtered, err := r.FilterEnabled([]string{"conn1"})
	if err != nil {
		t.Fatalf("erro inesperado ao filtrar: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Namespace() != "conn1" {
		t.Fatalf("esperava apenas conn1, obteve %v", filtered)
	}

	_, err = r.FilterEnabled([]string{"nao_existe"})
	if err == nil {
		t.Fatalf("esperava erro para conector inexistente")
	}

	tools := core.AllTools(filtered)
	if len(tools) != 1 || tools[0].Name != "conn1_ping" {
		t.Fatalf("esperava ferramenta conn1_ping")
	}

	tool, ok := core.FindTool(filtered, "conn1_ping")
	if !ok || tool.Name != "conn1_ping" {
		t.Fatalf("esperava encontrar conn1_ping")
	}
}
