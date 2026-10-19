package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	mcpbridge "connectors/internal/bridge/mcp"
	"connectors/internal/core"
)

type dummyConn struct{}

func (d *dummyConn) Namespace() string { return "test" }
func (d *dummyConn) Initialize(ctx context.Context) error { return nil }
func (d *dummyConn) Tools() []core.ToolDefinition {
	return []core.ToolDefinition{
		{
			Name:        "test_add",
			Description: "Soma dois inteiros",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}}}`),
			Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
				var p struct {
					A int `json:"a"`
					B int `json:"b"`
				}
				_ = json.Unmarshal(args, &p)
				return map[string]int{"result": p.A + p.B}, nil
			},
		},
	}
}
func (d *dummyConn) Close() error { return nil }

func TestBuildMCPServer(t *testing.T) {
	conn := &dummyConn{}
	wrapper := mcpbridge.BuildMCPServer([]core.Connector{conn})

	if wrapper == nil {
		t.Fatalf("wrapper retornado é nulo")
	}
	if wrapper.ToolCount != 1 {
		t.Errorf("esperava 1 ferramenta registrada, obteve %d", wrapper.ToolCount)
	}
	if wrapper.MCPServer == nil {
		t.Errorf("MCPServer interno é nulo")
	}
}
