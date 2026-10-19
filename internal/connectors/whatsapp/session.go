package whatsapp

import (
	"fmt"
	"os"
	"path/filepath"
)

// SessionConfig gerencia o caminho e configuração do banco SQLite de sessão local.
type SessionConfig struct {
	DBPath string
}

// DefaultSessionConfig localiza o arquivo em ~/.config/omni-connectors/whatsapp.db
func DefaultSessionConfig() (*SessionConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("impossível determinar diretório do usuário: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "omni-connectors")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório seguro de sessão: %w", err)
	}

	dbPath := filepath.Join(configDir, "whatsapp_session.db")
	return &SessionConfig{DBPath: dbPath}, nil
}
