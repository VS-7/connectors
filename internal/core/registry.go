package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Registry gerencia a coleção de conectores disponíveis no Hub.
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]Connector
}

// NewRegistry cria uma nova instância vazia do registro de conectores.
func NewRegistry() *Registry {
	return &Registry{
		connectors: make(map[string]Connector),
	}
}

// Register adiciona um conector ao catálogo pelo seu namespace.
func (r *Registry) Register(conn Connector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connectors[strings.ToLower(conn.Namespace())] = conn
}

// Get retorna um conector registrado pelo seu namespace.
func (r *Registry) Get(namespace string) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, ok := r.connectors[strings.ToLower(namespace)]
	return conn, ok
}

// All retorna todos os conectores atualmente registrados.
func (r *Registry) All() []Connector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Connector, 0, len(r.connectors))
	for _, c := range r.connectors {
		list = append(list, c)
	}
	return list
}

// FilterEnabled retorna apenas os conectores cujos namespaces constam na lista informada.
// Se a lista estiver vazia ou contiver "all" ou "*", retorna todos os conectores registrados.
func (r *Registry) FilterEnabled(enabled []string) ([]Connector, error) {
	if len(enabled) == 0 {
		return r.All(), nil
	}

	for _, name := range enabled {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed == "all" || trimmed == "*" {
			return r.All(), nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Connector
	var missing []string

	for _, name := range enabled {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed == "" {
			continue
		}
		if conn, ok := r.connectors[trimmed]; ok {
			result = append(result, conn)
		} else {
			missing = append(missing, trimmed)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("conector(es) desconhecido(s) solicitado(s): %s", strings.Join(missing, ", "))
	}

	return result, nil
}

// AllTools retorna a lista consolidada de todas as ferramentas de um conjunto de conectores.
func AllTools(connectors []Connector) []ToolDefinition {
	var tools []ToolDefinition
	for _, c := range connectors {
		tools = append(tools, c.Tools()...)
	}
	return tools
}

// FindTool busca uma ferramenta pelo nome exato dentro de uma lista de conectores.
func FindTool(connectors []Connector, toolName string) (ToolDefinition, bool) {
	for _, c := range connectors {
		for _, t := range c.Tools() {
			if t.Name == toolName {
				return t, true
			}
		}
	}
	return ToolDefinition{}, false
}

// InitializeConnectors inicializa todos os conectores da lista com o contexto informado.
func InitializeConnectors(ctx context.Context, connectors []Connector) error {
	for _, c := range connectors {
		if err := c.Initialize(ctx); err != nil {
			return fmt.Errorf("falha ao inicializar conector [%s]: %w", c.Namespace(), err)
		}
	}
	return nil
}

// CloseConnectors encerra graciosamente todos os conectores da lista.
func CloseConnectors(connectors []Connector) error {
	var errs []string
	for _, c := range connectors {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("[%s]: %v", c.Namespace(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("erros ao encerrar conectores: %s", strings.Join(errs, "; "))
	}
	return nil
}
