package whatsapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Limites do operador além das cotas da Guard.
const (
	maxOperatorMessage = 2000             // caracteres por mensagem
	duplicateWindow    = time.Hour        // mesmo texto para outro chat nesse intervalo é bloqueado
	minDuplicateLen    = 40               // textos curtos ("Bom dia!") podem se repetir
	sendMaxWait        = 90 * time.Second // espera máxima pela cota antes de recusar
)

// Erros do operador, para a ferramenta MCP explicar o bloqueio ao modelo.
var (
	ErrSendDisabled     = errors.New("envio desligado (OMNI_WA_SEND=off)")
	ErrColdRecipient    = errors.New("destinatário sem conversa prévia: o operador só responde chats em que a pessoa já te escreveu")
	ErrDuplicateMessage = errors.New("mesmo texto já enviado a outro chat na última hora (padrão de disparo em massa)")
)

// SendGuarded envia uma mensagem de texto em nome do usuário com as proteções anti-banimento:
// só para chats com mensagem recebida, sem disparo repetido, dentro da cota de envio e
// com "digitando…" proporcional ao tamanho, como uma pessoa faria.
func (c *Connector) SendGuarded(ctx context.Context, recipient, text string) (map[string]any, error) {
	if strings.EqualFold(os.Getenv("OMNI_WA_SEND"), "off") {
		return nil, ErrSendDisabled
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("mensagem vazia")
	}
	if n := utf8.RuneCountInString(text); n > maxOperatorMessage {
		return nil, fmt.Errorf("mensagem com %d caracteres; o limite é %d", n, maxOperatorMessage)
	}
	if c.waClient == nil || !c.waClient.IsConnected() {
		return nil, errors.New("whatsapp não conectado")
	}

	jid, err := types.ParseJID(normalizeChatID(recipient))
	if err != nil {
		return nil, fmt.Errorf("destinatário inválido: %w", err)
	}
	ok, err := c.msgStore.HasInbound(ctx, c.chatAliases(ctx, jid))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrColdRecipient
	}
	hash := textHash(text)
	if utf8.RuneCountInString(text) >= minDuplicateLen {
		dup, err := c.msgStore.SentElsewhere(ctx, hash, jid.String(), time.Now().Add(-duplicateWindow))
		if err != nil {
			return nil, err
		}
		if dup {
			return nil, ErrDuplicateMessage
		}
	}
	if err := c.guard.Reserve(ctx, ActionSend, sendMaxWait); err != nil {
		return nil, err
	}

	goOffline := c.simulateTyping(ctx, jid, text)
	resp, err := c.waClient.SendMessage(ctx, jid, &waE2E.Message{Conversation: proto.String(text)})
	goOffline()
	if err != nil {
		return nil, fmt.Errorf("enviar mensagem: %w", err)
	}

	if err := c.msgStore.RecordOperatorSend(ctx, jid.String(), hash, resp.Timestamp); err != nil {
		return nil, err
	}
	own := c.waClient.Store.GetJID().ToNonAD().String()
	if err := c.msgStore.SaveMessages(ctx, []Message{{
		ChatJID: jid.String(), ID: resp.ID, SenderJID: own, FromMe: true,
		Timestamp: resp.Timestamp, Kind: "text", Text: text,
	}}); err != nil {
		return nil, err
	}
	return map[string]any{
		"status":     "sent",
		"message_id": resp.ID,
		"timestamp":  resp.Timestamp.Format(time.RFC3339),
		"recipient":  jid.String(),
	}, nil
}

// simulateTyping mostra "digitando…" pelo tempo que uma pessoa levaria (~12 caracteres
// por segundo, entre 2 e 12 s) e devolve a função que volta a ficar offline depois do envio.
// Presença "available" fica ligada só nesse intervalo: com ela ativa o celular para de
// notificar. Falhas de presença são cosméticas e são ignoradas de propósito.
func (c *Connector) simulateTyping(ctx context.Context, jid types.JID, text string) (goOffline func()) {
	d := time.Duration(utf8.RuneCountInString(text)) * time.Second / 12
	d = min(max(d, 2*time.Second), 12*time.Second) + jitter(1500*time.Millisecond)
	_ = c.waClient.SendPresence(ctx, types.PresenceAvailable)
	_ = c.waClient.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	sleepCtx(ctx, d)
	_ = c.waClient.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)
	return func() {
		_ = c.waClient.SendPresence(context.WithoutCancel(ctx), types.PresenceUnavailable)
	}
}

// chatAliases devolve o JID e seu par telefone/LID, já que a mesma conversa pode estar
// gravada de um jeito ou de outro.
func (c *Connector) chatAliases(ctx context.Context, jid types.JID) []string {
	out := []string{jid.String()}
	lids := c.waClient.Store.LIDs
	switch jid.Server {
	case types.DefaultUserServer:
		if lid, err := lids.GetLIDForPN(ctx, jid); err == nil && !lid.IsEmpty() {
			out = append(out, lid.String())
		}
	case types.HiddenUserServer:
		if pn, err := lids.GetPNForLID(ctx, jid); err == nil && !pn.IsEmpty() {
			out = append(out, pn.String())
		}
	}
	return out
}

func textHash(text string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.Join(strings.Fields(text), " "))))
	return hex.EncodeToString(sum[:12])
}
