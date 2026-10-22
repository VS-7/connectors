package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// SessionManager gerencia a persistência de sessão e criptografia em SQLite.
type SessionManager struct {
	ConfigDir string
	DBPath    string
	Container *sqlstore.Container
}

// NewSessionManager conecta ou inicializa o banco SQLite em ~/.config/omni-connectors/whatsapp_session.db
func NewSessionManager() (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("impossível determinar diretório do usuário: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "omni-connectors")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório seguro de sessão: %w", err)
	}

	dbPath := filepath.Join(configDir, "whatsapp_session.db")
	// WAL + _txlock=immediate evitam SQLITE_BUSY nas escritas concorrentes logo após o pareamento.
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=10000&_txlock=immediate", dbPath)

	container, err := sqlstore.New(context.Background(), "sqlite3", dsn, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar banco SQLite do whatsmeow: %w", err)
	}

	return &SessionManager{
		ConfigDir: configDir,
		DBPath:    dbPath,
		Container: container,
	}, nil
}
