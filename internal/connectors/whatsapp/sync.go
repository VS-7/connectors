package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

// MessagesDBFile é o nome do banco de histórico, ao lado do banco de sessão.
const MessagesDBFile = "whatsapp_messages.db"

// SetPairingHandlers registra callbacks para exibir o QR code e, se phone não for vazio,
// o código de pareamento por número. Deve ser chamado antes de Initialize.
func (c *Connector) SetPairingHandlers(onQR func(code string), phone string, onPairCode func(code string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onQR = onQR
	c.pairPhone = phone
	c.onPairCode = onPairCode
}

// SetExclusive exige o lock da sessão em Initialize (sync, logout, daemon): se outro
// processo estiver conectado, Initialize falha em vez de cair para modo leitura.
func (c *Connector) SetExclusive(exclusive bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.exclusive = exclusive
}

// SetFullHistory define se o próximo pareamento pede o histórico completo ao celular.
// Deve ser chamado antes de Initialize.
func (c *Connector) SetFullHistory(full bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fullHistory = full
}

// ConfigDir devolve o diretório de configuração da sessão.
func (c *Connector) ConfigDir() string {
	if c.sessionMgr == nil {
		return ""
	}
	return c.sessionMgr.ConfigDir
}

// Stats devolve o progresso atual da sincronização.
func (c *Connector) Stats() SyncStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := c.stats
	s.Paired = c.isPaired
	if c.waClient != nil {
		s.Connected = c.waClient.IsConnected()
		s.Paired = s.Paired && c.waClient.Store.ID != nil
	}
	return s
}

// MessageStore expõe o histórico local para exportação.
func (c *Connector) MessageStore() *MessageStore {
	return c.msgStore
}

// RefreshGroups busca no servidor os grupos dos quais o usuário participa e grava nome,
// descrição e quantidade de participantes.
func (c *Connector) RefreshGroups(ctx context.Context) (int, error) {
	if c.waClient == nil || !c.waClient.IsConnected() {
		return 0, fmt.Errorf("whatsapp não conectado")
	}
	if err := c.guard.Reserve(ctx, ActionGroupRefresh, 0); err != nil {
		return 0, err
	}
	groups, err := c.waClient.GetJoinedGroups(ctx)
	if err != nil {
		return 0, fmt.Errorf("listar grupos: %w", err)
	}
	for _, g := range groups {
		if err := c.msgStore.UpdateGroupMeta(ctx, g.JID.String(), g.Name, g.Topic, len(g.Participants)); err != nil {
			return 0, err
		}
	}
	return len(groups), nil
}

// Logout desvincula este aparelho (some de "Aparelhos conectados" no celular) e apaga a sessão local.
func (c *Connector) Logout(ctx context.Context) error {
	if c.waClient == nil || c.waClient.Store.ID == nil {
		return fmt.Errorf("whatsapp não pareado")
	}
	if !c.waClient.WaitForConnection(30 * time.Second) {
		return fmt.Errorf("whatsapp não conectou a tempo para desvincular")
	}
	if err := c.waClient.Logout(ctx); err != nil {
		return fmt.Errorf("desvincular aparelho: %w", err)
	}
	c.mu.Lock()
	c.isPaired = false
	c.mu.Unlock()
	return nil
}

// ResetLocalSession apaga do SQLite um aparelho que o celular já desconectou, sem falar
// com o servidor (logout normal exige conexão, e o aparelho revogado só levaria a um 401).
// Exige o lock: não pode rodar com o daemon ativo.
func ResetLocalSession(ctx context.Context) (bool, error) {
	mgr, err := NewSessionManager()
	if err != nil {
		return false, err
	}
	defer mgr.Container.Close()
	lock, err := AcquireSessionLock(mgr.ConfigDir)
	if err != nil {
		return false, err
	}
	defer lock.Release()
	dev, err := mgr.Container.GetFirstDevice(ctx)
	if err != nil {
		return false, fmt.Errorf("carregar aparelho: %w", err)
	}
	if dev.ID == nil {
		return false, nil
	}
	if err := dev.Delete(ctx); err != nil {
		return false, fmt.Errorf("apagar aparelho local: %w", err)
	}
	return true, nil
}

// ReconcileArchived aplica o estado de arquivado do app state (fonte mais recente)
// sobre o que veio do histórico.
func (c *Connector) ReconcileArchived(ctx context.Context) (int, error) {
	if c.waClient == nil || c.waClient.Store.ID == nil {
		return 0, fmt.Errorf("whatsapp não pareado")
	}
	jids, err := c.msgStore.ChatJIDs(ctx)
	if err != nil {
		return 0, err
	}
	var archived int
	for _, raw := range jids {
		jid, err := types.ParseJID(raw)
		if err != nil {
			continue
		}
		settings, err := c.waClient.Store.ChatSettings.GetChatSettings(ctx, jid)
		if err != nil {
			slog.Warn("falha ao ler configurações do chat", "chat", raw, "error", err)
			continue
		}
		if !settings.Found {
			continue
		}
		if err := c.msgStore.SetArchived(ctx, raw, settings.Archived); err != nil {
			return archived, err
		}
		if settings.Archived {
			archived++
		}
	}
	return archived, nil
}

// ResolveName tenta achar um nome legível para um JID usando a sessão do cliente.
func (c *Connector) ResolveName(ctx context.Context, rawJID string) string {
	if c.waClient == nil {
		return ""
	}
	return ResolveName(ctx, c.waClient.Store, rawJID)
}

// ResolveName tenta achar um nome legível para um JID (telefone ou LID) na agenda
// e nos push names sincronizados, sem precisar de conexão ativa.
func ResolveName(ctx context.Context, dev *store.Device, rawJID string) string {
	if dev == nil || rawJID == "" {
		return ""
	}
	jid, err := types.ParseJID(rawJID)
	if err != nil {
		return ""
	}
	lookup := func(j types.JID) string {
		info, err := dev.Contacts.GetContact(ctx, j)
		if err != nil || !info.Found {
			return ""
		}
		for _, n := range []string{info.FullName, info.FirstName, info.PushName, info.BusinessName} {
			if n != "" {
				return n
			}
		}
		return ""
	}
	if name := lookup(jid); name != "" {
		return name
	}
	if jid.Server == types.HiddenUserServer {
		pn, err := dev.LIDs.GetPNForLID(ctx, jid)
		if err == nil && !pn.IsEmpty() {
			if name := lookup(pn); name != "" {
				return name
			}
			return "+" + pn.User
		}
		return ""
	}
	return "+" + jid.User
}
