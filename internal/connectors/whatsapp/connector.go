package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"connectors/internal/core"
	"connectors/pkg/jsonschema"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// Connector implementa core.Connector com suporte nativo a whatsmeow (SQLite local)
// e fallback opcional via Evolution API caso configurada no ambiente.
type Connector struct {
	mu         sync.RWMutex
	sessionMgr *SessionManager
	waClient   *whatsmeow.Client
	msgStore   *MessageStore
	guard      *Guard // cotas anti-banimento para tudo que vai ao WhatsApp
	qrCode     string
	isPaired   bool
	stats      SyncStats

	// Callbacks opcionais de pareamento, usados pelo CLI omni-wa.
	onQR       func(code string)
	onPairCode func(code string)
	pairPhone  string

	// Só um processo pode ficar conectado com a sessão. Sem o lock, o conector fica em
	// modo leitura (consulta o SQLite, não conecta). exclusive exige o lock.
	lock      *SessionLock
	exclusive bool
	readOnly  bool
	// fullHistory pede ao celular o histórico completo no pareamento (pesado para o celular;
	// desnecessário quando o histórico já está no SQLite).
	fullHistory bool

	// Sinais entre o handler de eventos e os workers do daemon.
	mediaWake      chan struct{}
	transcribeWake chan struct{}
	onDemand       chan string
	loggedOut      chan struct{}
	loggedOutOnce  sync.Once

	// Configuração opcional de Evolution API REST
	apiURL     string
	apiKey     string
	instance   string
	httpClient *http.Client
}

// NewConnector instancia o conector WhatsApp.
func NewConnector() *Connector {
	return &Connector{
		apiURL:         os.Getenv("EVOLUTION_API_URL"),
		apiKey:         os.Getenv("EVOLUTION_API_KEY"),
		instance:       os.Getenv("EVOLUTION_INSTANCE"),
		httpClient:     &http.Client{Timeout: 15 * time.Second},
		mediaWake:      make(chan struct{}, 1),
		transcribeWake: make(chan struct{}, 1),
		onDemand:       make(chan string, 64),
		loggedOut:      make(chan struct{}),
	}
}

// Namespace identifica o conector.
func (c *Connector) Namespace() string {
	return "whatsapp"
}

// Initialize inicializa a sessão SQLite local do whatsmeow e conecta ao WebSocket se pareado.
func (c *Connector) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Se não for Evolution API pura, inicializa o whatsmeow nativo com SQLite
	mgr, err := NewSessionManager()
	if err != nil {
		slog.Warn("não foi possível inicializar SQLite local do whatsmeow", "error", err)
	} else {
		c.sessionMgr = mgr
		deviceStore, err := mgr.Container.GetFirstDevice(ctx)
		if err != nil {
			return fmt.Errorf("falha ao carregar dispositivo do SQLite: %w", err)
		}

		msgStore, err := OpenMessageStore(ctx, filepath.Join(mgr.ConfigDir, MessagesDBFile))
		if err != nil {
			return fmt.Errorf("abrir histórico local do whatsapp: %w", err)
		}
		c.msgStore = msgStore
		c.guard = NewGuard(msgStore, DefaultPolicies())

		// OMNI_WA_READONLY=on (servidores MCP das sessões do Claude): nunca conecta, mesmo com a
		// sessão livre. Só o daemon fala com o WhatsApp; abrir uma conexão por sessão do Claude
		// seria o vai-e-vem de logins que queremos evitar.
		if os.Getenv("OMNI_WA_READONLY") == "on" && !c.exclusive {
			c.readOnly = true
		} else {
			lock, err := AcquireSessionLock(mgr.ConfigDir)
			switch {
			case err == nil:
				c.lock = lock
			case errors.Is(err, ErrSessionBusy) && !c.exclusive:
				c.readOnly = true
				slog.Info("whatsapp em modo leitura: a sessão está com outro processo (daemon omni-wa)", "detalhe", err)
			default:
				return err
			}
		}

		// Histórico completo só quando pedido (vale apenas para novos pareamentos).
		store.DeviceProps.RequireFullSync = proto.Bool(c.fullHistory)
		store.SetOSInfo("Omni Connectors", [3]uint32{1, 0, 0})

		setupSignalLogger()
		c.waClient = whatsmeow.NewClient(deviceStore, newWALogger(os.Getenv("WHATSMEOW_LOG_LEVEL")))
		// Emite eventos de arquivamento também na sincronização completa do app state.
		c.waClient.EmitAppStateEventsOnFullSync = true
		c.waClient.AddEventHandler(c.handleEvent)

		if c.readOnly {
			c.isPaired = deviceStore.ID != nil
			return nil
		}
		if deviceStore.ID != nil {
			c.isPaired = true
			slog.Info("whatsapp pareado localmente no SQLite, conectando WebSocket...")
			go func() {
				if err := c.waClient.Connect(); err != nil {
					slog.Warn("erro ao conectar whatsmeow em background", "error", err)
				}
			}()
		} else {
			c.isPaired = false
			slog.Info("whatsapp não pareado no SQLite. Disponibilizando canal de QR code.")
			go c.listenQR(ctx)
		}
	}

	return nil
}

// maxQRRounds limita quantas vezes o canal de QR é reaberto após expirar sem leitura.
const maxQRRounds = 5

func (c *Connector) listenQR(ctx context.Context) {
	for round := 1; round <= maxQRRounds; round++ {
		paired, err := c.qrRound(ctx)
		if err != nil {
			slog.Warn("erro no pareamento whatsmeow", "error", err)
			return
		}
		if paired || ctx.Err() != nil {
			return
		}
		slog.Info("QR code expirou sem leitura, gerando novos códigos", "rodada", round+1)
		c.waClient.Disconnect()
	}
	slog.Warn("pareamento abandonado após várias rodadas de QR sem leitura")
}

// qrRound abre um canal de QR, conecta e consome os eventos até sucesso ou expiração.
func (c *Connector) qrRound(ctx context.Context) (bool, error) {
	qrChan, err := c.waClient.GetQRChannel(ctx)
	if err != nil {
		return false, fmt.Errorf("abrir canal de QR: %w", err)
	}
	if err := c.waClient.Connect(); err != nil {
		return false, fmt.Errorf("conectar para QR: %w", err)
	}

	c.mu.RLock()
	phone, onPairCode := c.pairPhone, c.onPairCode
	c.mu.RUnlock()
	if phone != "" {
		code, err := c.waClient.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Mac OS)")
		if err != nil {
			return false, fmt.Errorf("gerar código de pareamento por telefone: %w", err)
		}
		if onPairCode != nil {
			onPairCode(code)
		}
	}

	for evt := range qrChan {
		c.mu.Lock()
		onQR := c.onQR
		switch evt.Event {
		case whatsmeow.QRChannelEventCode:
			c.qrCode = evt.Code
			slog.Info("novo QR code de pareamento gerado para o WhatsApp")
		case whatsmeow.QRChannelSuccess.Event:
			c.isPaired = true
			c.qrCode = ""
			slog.Info("WhatsApp pareado com sucesso via whatsmeow!")
		}
		c.mu.Unlock()

		switch evt.Event {
		case whatsmeow.QRChannelEventCode:
			if onQR != nil {
				onQR(evt.Code)
			}
		case whatsmeow.QRChannelSuccess.Event:
			return true, nil
		case whatsmeow.QRChannelTimeout.Event:
			return false, nil
		case whatsmeow.QRChannelEventError:
			return false, fmt.Errorf("pareamento recusado: %w", evt.Error)
		}
	}
	return false, nil
}

// Tools lista as ferramentas disponíveis para LLMs.
func (c *Connector) Tools() []core.ToolDefinition {
	statusSchema := jsonschema.NewObjectSchema().RawMessage()

	sendMsgSchema := jsonschema.NewObjectSchema().
		AddProperty("recipient", "string", "Número do destinatário com DDI e DDD (ex: '5511999998888') ou JID de grupo", true).
		AddProperty("message", "string", "Texto da mensagem a ser enviada", true).
		RawMessage()

	getRecentSchema := jsonschema.NewObjectSchema().
		AddProperty("chat_id", "string", "ID do chat ou número de telefone para buscar histórico", true).
		AddProperty("limit", "integer", "Quantidade máxima de mensagens a retornar (padrão 10)", false).
		RawMessage()

	searchChatsSchema := jsonschema.NewObjectSchema().
		AddProperty("query", "string", "Termo de busca para localizar contatos ou conversas", true).
		RawMessage()

	listChatsSchema := jsonschema.NewObjectSchema().
		AddProperty("archived_only", "boolean", "Apenas chats arquivados", false).
		AddProperty("groups_only", "boolean", "Apenas grupos", false).
		AddProperty("limit", "integer", "Quantidade máxima de chats (padrão 50)", false).
		RawMessage()

	activitySchema := jsonschema.NewObjectSchema().
		AddProperty("since_hours", "number", "Janela em horas a partir de agora (padrão 24)", false).
		AddProperty("skip_archived", "boolean", "Ignora chats arquivados", false).
		AddProperty("limit", "integer", "Máximo de mensagens, ficando com as mais recentes (padrão 300, máx. 1000)", false).
		RawMessage()

	searchMsgsSchema := jsonschema.NewObjectSchema().
		AddProperty("query", "string", "Termos que precisam aparecer todos no texto ou na transcrição do áudio", true).
		AddProperty("chat_id", "string", "Restringe a busca a um chat (JID ou número)", false).
		AddProperty("limit", "integer", "Máximo de resultados, dos mais recentes (padrão 30, máx. 200)", false).
		RawMessage()

	return []core.ToolDefinition{
		{
			Name:        "whatsapp_list_chats",
			Description: "Lista chats sincronizados localmente com contagem de mensagens e período, filtrando por arquivados e/ou grupos.",
			InputSchema: listChatsSchema,
			Handler:     c.handleListChats,
		},
		{
			Name:        "whatsapp_get_connection_status",
			Description: "Retorna o status da conexão do WhatsApp (conectado, pareado, ou código QR para escaneamento).",
			InputSchema: statusSchema,
			Handler:     c.handleGetStatus,
		},
		{
			Name:        "whatsapp_send_message",
			Description: "Envia uma mensagem de texto no WhatsApp em nome do usuário (operador). Proteções anti-banimento: só para chats em que a pessoa já escreveu, até 10 por hora e 40 por dia, das 9h às 21h, e o mesmo texto não pode ir para vários chats. Mostre o texto ao usuário e peça confirmação antes de enviar.",
			InputSchema: sendMsgSchema,
			Handler:     c.handleSendMessage,
		},
		{
			Name:        "whatsapp_get_recent_messages",
			Description: "Busca as mensagens mais recentes de um chat no WhatsApp.",
			InputSchema: getRecentSchema,
			Handler:     c.handleGetRecentMessages,
		},
		{
			Name:        "whatsapp_search_chats",
			Description: "Pesquisa chats e contatos no WhatsApp pelo nome ou número.",
			InputSchema: searchChatsSchema,
			Handler:     c.handleSearchChats,
		},
		{
			Name:        "whatsapp_recent_activity",
			Description: "Tudo o que aconteceu no WhatsApp nas últimas horas, em todos os chats, agrupado por conversa (mais recente primeiro). Áudios vêm transcritos. Lê só o histórico local, sem tráfego no WhatsApp.",
			InputSchema: activitySchema,
			Handler:     c.handleRecentActivity,
		},
		{
			Name:        "whatsapp_search_messages",
			Description: "Busca mensagens pelo conteúdo (texto e transcrição de áudios) em todos os chats ou em um só. Lê só o histórico local, sem tráfego no WhatsApp.",
			InputSchema: searchMsgsSchema,
			Handler:     c.handleSearchMessages,
		},
	}
}

// Close desconecta o cliente e encerra o SQLite.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.waClient != nil && c.waClient.IsConnected() {
		c.waClient.Disconnect()
	}
	if c.sessionMgr != nil && c.sessionMgr.Container != nil {
		_ = c.sessionMgr.Container.Close()
	}
	if c.msgStore != nil {
		if err := c.msgStore.Close(); err != nil {
			return fmt.Errorf("fechar histórico local: %w", err)
		}
	}
	if err := c.lock.Release(); err != nil {
		return fmt.Errorf("liberar lock da sessão: %w", err)
	}
	return nil
}

func (c *Connector) handleGetStatus(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := map[string]any{
		"storage":     "sqlite",
		"db_path":     "",
		"is_paired":   c.isPaired,
		"connected":   false,
		"has_qr_code": c.qrCode != "",
		"history":     c.stats,
		"read_only":   c.readOnly,
	}

	if c.sessionMgr != nil {
		status["db_path"] = c.sessionMgr.DBPath
	}

	if c.waClient != nil {
		status["connected"] = c.waClient.IsConnected()
		if c.waClient.Store != nil && c.waClient.Store.ID != nil {
			status["user_jid"] = c.waClient.Store.ID.String()
		}
	}
	// Em modo leitura quem conecta é o daemon: sem ele, o histórico local para no tempo.
	if c.readOnly && c.sessionMgr != nil {
		if pid, ok := daemonAlive(c.sessionMgr.ConfigDir); ok {
			status["daemon"] = fmt.Sprintf("ativo (pid %d)", pid)
			status["connected"] = true
		} else {
			status["daemon"] = "parado: o histórico não está sendo atualizado (scripts/omni-wa-agent.sh install)"
		}
	}
	if c.msgStore != nil {
		if last, err := c.msgStore.LastMessageAt(ctx); err == nil && !last.IsZero() {
			status["last_message_at"] = last.Format(time.RFC3339)
		}
	}

	if c.qrCode != "" {
		status["qr_code_raw"] = c.qrCode
		status["instruction"] = "Utilize o WhatsApp no celular: Menu > Aparelhos Conectados > Conectar um aparelho e aponte para o código."
	}

	return status, nil
}

func (c *Connector) handleSendMessage(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Recipient string `json:"recipient"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Recipient == "" || args.Message == "" {
		return nil, fmt.Errorf("recipient e message são obrigatórios")
	}

	cleanNumber := strings.TrimPrefix(args.Recipient, "+")
	cleanNumber = strings.ReplaceAll(cleanNumber, "-", "")
	cleanNumber = strings.ReplaceAll(cleanNumber, " ", "")

	// 1. O daemon está com a sessão: encaminha pelo socket local (mesmas proteções).
	if c.readOnly {
		return SendViaDaemon(ctx, c.sessionMgr.ConfigDir, args.Recipient, args.Message)
	}

	// 2. Sessão nativa neste processo, com as proteções do operador.
	if c.waClient != nil && c.waClient.IsConnected() {
		return c.SendGuarded(ctx, args.Recipient, args.Message)
	}

	// 3. Fallback via Evolution API se configurada
	if c.apiURL != "" && c.instance != "" {
		endpoint := fmt.Sprintf("%s/message/sendText/%s", strings.TrimRight(c.apiURL, "/"), c.instance)
		payload := map[string]any{
			"number":  cleanNumber,
			"text":    args.Message,
			"options": map[string]any{"delay": 1000},
		}
		bodyBytes, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("falha ao enviar via WhatsApp gateway: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			errBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("erro no envio WhatsApp (%d): %s", resp.StatusCode, string(errBytes))
		}

		return map[string]any{
			"status":    "sent_via_evolution_api",
			"recipient": cleanNumber,
			"timestamp": time.Now().Format(time.RFC3339),
			"snippet":   core.TruncateText(args.Message, 100),
		}, nil
	}

	// Sem conexão e sem gateway: nada foi enviado.
	return nil, fmt.Errorf("whatsapp não conectado: rode omni-wa sync para parear ou instale o daemon (scripts/omni-wa-agent.sh install)")
}

func (c *Connector) handleGetRecentMessages(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		ChatID string `json:"chat_id"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.ChatID == "" {
		return nil, fmt.Errorf("chat_id é obrigatório")
	}
	if args.Limit <= 0 {
		args.Limit = 10
	}
	if c.msgStore == nil {
		return nil, fmt.Errorf("histórico local do whatsapp indisponível")
	}

	chatJID := normalizeChatID(args.ChatID)
	msgs, err := c.msgStore.Messages(ctx, chatJID, args.Limit)
	if err != nil {
		return nil, err
	}

	names := map[string]string{}
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, c.messageView(ctx, m, names))
	}
	return map[string]any{"chat_id": chatJID, "messages": out}, nil
}

func (c *Connector) handleRecentActivity(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		SinceHours   float64 `json:"since_hours"`
		SkipArchived bool    `json:"skip_archived"`
		Limit        int     `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.SinceHours <= 0 {
		args.SinceHours = 24
	}
	if args.Limit <= 0 {
		args.Limit = 300
	}
	args.Limit = min(args.Limit, 1000)
	if c.msgStore == nil {
		return nil, fmt.Errorf("histórico local do whatsapp indisponível")
	}

	since := time.Now().Add(-time.Duration(args.SinceHours * float64(time.Hour)))
	msgs, err := c.msgStore.MessagesSince(ctx, since, args.Limit)
	if err != nil {
		return nil, err
	}
	chats, err := c.msgStore.Chats(ctx)
	if err != nil {
		return nil, err
	}

	type chatActivity struct {
		jid  string
		last time.Time
		msgs []map[string]any
	}
	byChat := map[string]*chatActivity{}
	names := map[string]string{}
	kept := 0
	for _, m := range msgs {
		if args.SkipArchived && chats[m.ChatJID].Archived {
			continue
		}
		a := byChat[m.ChatJID]
		if a == nil {
			a = &chatActivity{jid: m.ChatJID}
			byChat[m.ChatJID] = a
		}
		a.msgs = append(a.msgs, c.messageView(ctx, m, names))
		a.last = m.Timestamp
		kept++
	}
	ordered := slices.SortedFunc(maps.Values(byChat), func(a, b *chatActivity) int {
		return b.last.Compare(a.last)
	})

	out := make([]map[string]any, 0, len(ordered))
	for _, a := range ordered {
		ch := chats[a.jid]
		out = append(out, map[string]any{
			"chat_id":  a.jid,
			"name":     c.chatName(ctx, a.jid, chats),
			"group":    ch.IsGroup,
			"archived": ch.Archived,
			"messages": a.msgs,
		})
	}
	return map[string]any{
		"since":     since.Format(time.RFC3339),
		"messages":  kept,
		"truncated": len(msgs) == args.Limit,
		"chats":     out,
	}, nil
}

func (c *Connector) handleSearchMessages(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Query  string `json:"query"`
		ChatID string `json:"chat_id"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return nil, fmt.Errorf("query é obrigatória")
	}
	if args.Limit <= 0 {
		args.Limit = 30
	}
	args.Limit = min(args.Limit, 200)
	if c.msgStore == nil {
		return nil, fmt.Errorf("histórico local do whatsapp indisponível")
	}

	chatJID := ""
	if args.ChatID != "" {
		chatJID = normalizeChatID(args.ChatID)
	}
	msgs, err := c.msgStore.SearchMessages(ctx, args.Query, chatJID, args.Limit)
	if err != nil {
		return nil, err
	}
	chats, err := c.msgStore.Chats(ctx)
	if err != nil {
		return nil, err
	}

	names := map[string]string{}
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		v := c.messageView(ctx, m, names)
		v["chat_id"] = m.ChatJID
		v["chat"] = c.chatName(ctx, m.ChatJID, chats)
		out = append(out, v)
	}
	return map[string]any{"query": args.Query, "results": out}, nil
}

// messageView reduz uma mensagem ao que interessa ao modelo: autor legível, horário e texto
// (áudio vem com a transcrição). names guarda os nomes já resolvidos na mesma chamada.
func (c *Connector) messageView(ctx context.Context, m Message, names map[string]string) map[string]any {
	from := m.SenderName
	if m.FromMe {
		from = "eu"
	} else if from == "" {
		var ok bool
		if from, ok = names[m.SenderJID]; !ok {
			from = c.ResolveName(ctx, m.SenderJID)
			names[m.SenderJID] = from
		}
	}
	text := m.Text
	if m.Kind == "audio" && m.Transcript != "" {
		text = "[áudio " + m.Text + "] " + m.Transcript
	}
	return map[string]any{
		"from":      from,
		"timestamp": m.Timestamp.Format(time.RFC3339),
		"kind":      m.Kind,
		"text":      core.TruncateText(text, 500),
	}
}

// chatName devolve o nome do grupo ou, para conversas individuais sem nome salvo, o
// contato da agenda.
func (c *Connector) chatName(ctx context.Context, jid string, chats map[string]Chat) string {
	if ch, ok := chats[jid]; ok && ch.Name != "" {
		return ch.Name
	}
	if strings.HasSuffix(jid, "@"+types.GroupServer) {
		return ""
	}
	return c.ResolveName(ctx, jid)
}

func (c *Connector) handleSearchChats(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Query == "" {
		return nil, fmt.Errorf("query é obrigatória")
	}
	if c.msgStore == nil {
		return nil, fmt.Errorf("histórico local do whatsapp indisponível")
	}

	chats, err := c.msgStore.ListChats(ctx, ChatFilter{Query: args.Query, Limit: 10})
	if err != nil {
		return nil, err
	}
	return map[string]any{"query": args.Query, "chats": compactChats(chats)}, nil
}

func (c *Connector) handleListChats(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		ArchivedOnly bool `json:"archived_only"`
		GroupsOnly   bool `json:"groups_only"`
		Limit        int  `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Limit <= 0 {
		args.Limit = 50
	}
	if c.msgStore == nil {
		return nil, fmt.Errorf("histórico local do whatsapp indisponível")
	}

	chats, err := c.msgStore.ListChats(ctx, ChatFilter{
		ArchivedOnly: args.ArchivedOnly,
		GroupsOnly:   args.GroupsOnly,
		Limit:        args.Limit,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"count": len(chats), "chats": compactChats(chats)}, nil
}

// normalizeChatID aceita JID completo ou número de telefone.
func normalizeChatID(id string) string {
	if strings.Contains(id, "@") {
		return id
	}
	number := strings.NewReplacer("+", "", "-", "", " ", "").Replace(id)
	return types.NewJID(number, types.DefaultUserServer).String()
}

func compactChats(chats []ChatSummary) []map[string]any {
	out := make([]map[string]any, 0, len(chats))
	for _, cs := range chats {
		item := map[string]any{
			"id":       cs.JID,
			"name":     cs.Name,
			"group":    cs.IsGroup,
			"archived": cs.Archived,
			"messages": cs.Messages,
		}
		if !cs.LastAt.IsZero() {
			item["last_at"] = cs.LastAt.Format(time.DateOnly)
		}
		out = append(out, item)
	}
	return out
}
