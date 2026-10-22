// Comando omni-wa pareia o WhatsApp, sincroniza o histórico para o SQLite local
// e exporta conversas em Markdown para análise.
//
//	omni-wa sync   [-idle 8m] [-max 2h] [-qr-dir data/whatsapp] [-phone 5511...] [-open] [-full]
//	omni-wa chats  [-archived] [-groups]
//	omni-wa export [-archived] [-groups] [-out data/whatsapp/export]
//	omni-wa logout
//	omni-wa daemon                      conexão contínua + chaves, download e transcrição de áudios
//	omni-wa transcribe [-limit 50]      transcreve áudios já baixados (não precisa de conexão)
//	omni-wa status                      quem está com a sessão, cotas usadas e andamento dos áudios
//	omni-wa reset                       apaga localmente um aparelho já desconectado pelo celular
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
	"unicode"

	"connectors/internal/connectors/whatsapp"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/text/unicode/norm"
)

func main() {
	// WHATSMEOW_LOG_LEVEL=DEBUG liga o debug do whatsmeow e, junto, o do próprio omni-wa.
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("WHATSMEOW_LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: omni-wa <sync|chats|export|logout|daemon|transcribe|status> [flags]")
		os.Exit(2)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var err error
	switch os.Args[1] {
	case "sync":
		err = runSync(ctx, os.Args[2:])
	case "chats":
		err = runChats(ctx, os.Args[2:])
	case "export":
		err = runExport(ctx, os.Args[2:])
	case "logout":
		err = runLogout(ctx)
	case "daemon":
		err = runDaemon(ctx)
	case "transcribe":
		err = runTranscribe(ctx, os.Args[2:])
	case "status":
		err = runStatus(ctx)
	case "reset":
		err = runReset(ctx)
	default:
		err = fmt.Errorf("comando desconhecido %q", os.Args[1])
	}
	if err != nil {
		slog.Error("omni-wa falhou", "error", err)
		os.Exit(1)
	}
}

// runDaemon sai com sucesso (sem erro) quando reiniciar não resolveria: sem pareamento,
// aparelho desconectado pelo celular ou sessão já em uso. Assim o launchd (KeepAlive
// só em falha) não entra em loop; o motivo fica no log.
func runDaemon(ctx context.Context) error {
	conn := whatsapp.NewConnector()
	conn.SetExclusive(true)
	if err := conn.Initialize(ctx); err != nil {
		if errors.Is(err, whatsapp.ErrSessionBusy) {
			slog.Error("daemon não iniciado", "motivo", err)
			return nil
		}
		return err
	}
	defer conn.Close()
	if !conn.Stats().Paired {
		slog.Error("daemon não iniciado: whatsapp não pareado; rode omni-wa sync")
		return nil
	}

	cfg := whatsapp.DaemonConfig{
		NewTranscriber: func() (*whatsapp.Transcriber, error) {
			return whatsapp.NewTranscriberFromEnv(conn.ConfigDir())
		},
		// Desligado por padrão: pedir histórico e reenvio de mídia faz o celular trabalhar,
		// o que um cliente normal não faz em volume. OMNI_WA_BACKLOG=on liga, devagar e de dia.
		Backlog: strings.EqualFold(os.Getenv("OMNI_WA_BACKLOG"), "on"),
		// OMNI_WA_SEND=off: sem operador (o socket de envio nem abre).
		ControlSocket: !strings.EqualFold(os.Getenv("OMNI_WA_SEND"), "off"),
	}
	err := conn.RunDaemon(ctx, cfg)
	if errors.Is(err, whatsapp.ErrLoggedOut) {
		slog.Error("daemon encerrado", "motivo", err)
		return nil
	}
	return err
}

func runTranscribe(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("transcribe", flag.ExitOnError)
	limit := fs.Int("limit", 50, "máximo de áudios a transcrever nesta execução")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	mgr, err := whatsapp.NewSessionManager()
	if err != nil {
		return err
	}
	defer mgr.Container.Close()
	t, err := whatsapp.NewTranscriberFromEnv(mgr.ConfigDir)
	if err != nil {
		return err
	}
	ms, err := whatsapp.OpenMessageStore(ctx, filepath.Join(mgr.ConfigDir, whatsapp.MessagesDBFile))
	if err != nil {
		return err
	}
	defer ms.Close()

	done := 0
	for done < *limit {
		n, err := whatsapp.TranscribePending(ctx, ms, t, min(10, *limit-done))
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		done += n
		slog.Info("transcrevendo", "processados", done)
	}
	slog.Info("transcrição concluída", "processados", done)
	return nil
}

func runStatus(ctx context.Context) error {
	mgr, err := whatsapp.NewSessionManager()
	if err != nil {
		return err
	}
	defer mgr.Container.Close()
	ms, err := whatsapp.OpenMessageStore(ctx, filepath.Join(mgr.ConfigDir, whatsapp.MessagesDBFile))
	if err != nil {
		return err
	}
	defer ms.Close()

	if pid := whatsapp.SessionHolder(mgr.ConfigDir); pid != "" {
		fmt.Printf("sessão: conectada pelo processo %s (daemon)\n", pid)
	} else {
		fmt.Println("sessão: livre (nenhum processo conectado)")
	}
	policies := whatsapp.DefaultPolicies()
	usage, err := whatsapp.NewGuard(ms, policies).Usage(ctx)
	if err != nil {
		return err
	}
	fmt.Println("cotas (últimas 24h):")
	for _, a := range []string{whatsapp.ActionSend, whatsapp.ActionMediaRecent, whatsapp.ActionMediaBacklog,
		whatsapp.ActionMediaRetry, whatsapp.ActionHistoryRequest, whatsapp.ActionGroupRefresh} {
		fmt.Printf("  %-16s %3d / %d\n", a, usage[a], policies[a].PerDay)
	}

	stats, err := ms.AudioStats(ctx)
	if err != nil {
		return err
	}
	total := 0
	for _, n := range stats {
		total += n
	}
	fmt.Printf("áudios: %d\n", total)
	for _, k := range []string{"sem_chave", whatsapp.MediaKeysUnavailable, "na_fila_download", whatsapp.MediaExpired,
		whatsapp.MediaRetryRequested, whatsapp.MediaDownloaded, whatsapp.MediaTranscribed, whatsapp.MediaFailed} {
		if stats[k] > 0 {
			fmt.Printf("  %-18s %d\n", k, stats[k])
		}
	}
	return nil
}

func runReset(ctx context.Context) error {
	removed, err := whatsapp.ResetLocalSession(ctx)
	if err != nil {
		return err
	}
	if removed {
		slog.Info("aparelho local apagado; rode omni-wa sync para parear de novo")
	} else {
		slog.Info("nenhum aparelho local para apagar")
	}
	return nil
}

func runLogout(ctx context.Context) error {
	conn := whatsapp.NewConnector()
	conn.SetExclusive(true)
	if err := conn.Initialize(ctx); err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.Logout(ctx); err != nil {
		return err
	}
	slog.Info("aparelho desvinculado do WhatsApp e sessão local apagada")
	return nil
}

func runSync(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	idle := fs.Duration("idle", 8*time.Minute, "encerra após esse tempo sem novos lotes de histórico")
	maxDur := fs.Duration("max", 2*time.Hour, "tempo máximo total de sincronização")
	qrDir := fs.String("qr-dir", "data/whatsapp", "pasta onde o QR code (qr.html) é gravado")
	phone := fs.String("phone", "", "número com DDI para parear por código em vez de QR (ex: 5511999998888)")
	openBrowser := fs.Bool("open", true, "abre a página do QR no navegador")
	full := fs.Bool("full", false, "pede o histórico completo ao celular (pesado; só no primeiro pareamento)")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := os.MkdirAll(*qrDir, 0o700); err != nil {
		return fmt.Errorf("criar pasta do QR: %w", err)
	}
	page := filepath.Join(*qrDir, "qr.html")

	opened := false
	onQR := func(code string) {
		png, err := qrcode.Encode(code, qrcode.Medium, 384)
		if err != nil {
			slog.Warn("falha ao gerar imagem do QR", "error", err)
			return
		}
		body := fmt.Sprintf(`<img src="data:image/png;base64,%s" alt="QR WhatsApp">
<p>WhatsApp no celular → Configurações → Aparelhos conectados → Conectar um aparelho.</p>
<p class="muted">O código renova sozinho a cada ~20s.</p>`, base64.StdEncoding.EncodeToString(png))
		if err := writePage(page, body, true); err != nil {
			slog.Warn("falha ao gravar página do QR", "error", err)
			return
		}
		slog.Info("novo QR code disponível", "pagina", page)
		if *openBrowser && !opened {
			opened = true
			if err := exec.Command("open", page).Start(); err != nil {
				slog.Warn("não foi possível abrir o navegador", "error", err)
			}
		}
	}
	onPairCode := func(code string) {
		slog.Info("código de pareamento por telefone", "codigo", code)
		body := fmt.Sprintf(`<p>Código de pareamento:</p><p class="code">%s</p>
<p>WhatsApp → Aparelhos conectados → Conectar um aparelho → Conectar com número de telefone.</p>`, html.EscapeString(code))
		if err := writePage(page, body, true); err != nil {
			slog.Warn("falha ao gravar página do código", "error", err)
		}
	}

	conn := whatsapp.NewConnector()
	conn.SetExclusive(true)
	conn.SetFullHistory(*full)
	conn.SetPairingHandlers(onQR, *phone, onPairCode)
	if err := conn.Initialize(ctx); err != nil {
		return err
	}
	defer conn.Close()

	start := time.Now()
	var connectedAt time.Time
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("interrompido, finalizando sincronização")
			return finishSync(context.Background(), conn)
		case <-ticker.C:
		}

		st := conn.Stats()
		if st.Paired && st.Connected && connectedAt.IsZero() {
			// O app state (arquivados) chega sozinho: o whatsmeow busca ao receber as chaves.
			connectedAt = time.Now()
		}
		slog.Info("sincronização",
			"pareado", st.Paired, "conectado", st.Connected, "lotes", st.HistoryChunks,
			"conversas", st.Conversations, "mensagens", st.Messages, "progresso", st.Progress)

		if !connectedAt.IsZero() && !st.Paired {
			_ = writePage(page, `<p>Aparelho desconectado pelo celular. Rode o sync de novo para gerar outro QR.</p>`, false) //nolint:errcheck // página é só informativa
			return fmt.Errorf("aparelho desconectado pelo celular durante a sincronização")
		}
		if time.Since(start) > *maxDur {
			slog.Warn("tempo máximo atingido")
			return finishSync(ctx, conn)
		}
		if connectedAt.IsZero() {
			continue
		}
		progress := fmt.Sprintf(`<p class="ok">✓ WhatsApp pareado</p>
<p><b>Mantenha o WhatsApp aberto e na tela do celular</b> até os lotes pararem de chegar.</p>
<p>Lotes de histórico: <b>%d</b> · conversas: <b>%d</b> · mensagens: <b>%d</b> · progresso: <b>%d%%</b></p>
<p class="muted">Atualizado %s</p>`, st.HistoryChunks, st.Conversations, st.Messages, st.Progress, time.Now().Format("15:04:05"))
		if err := writePage(page, progress, true); err != nil {
			slog.Warn("falha ao atualizar página", "error", err)
		}
		last := connectedAt
		if st.LastHistoryAt.After(last) {
			last = st.LastHistoryAt
		}
		if time.Since(last) > *idle {
			slog.Info("sem novos lotes de histórico, encerrando", "ocioso", time.Since(last).Round(time.Second))
			return finishSync(ctx, conn)
		}
	}
}

func finishSync(ctx context.Context, conn *whatsapp.Connector) error {
	groups, err := conn.RefreshGroups(ctx)
	if err != nil {
		slog.Warn("não foi possível atualizar metadados dos grupos", "error", err)
	}
	archived, err := conn.ReconcileArchived(ctx)
	if err != nil {
		slog.Warn("não foi possível reconciliar arquivados", "error", err)
	}
	st := conn.Stats()
	slog.Info("sincronização concluída", "grupos", groups, "arquivados_app_state", archived,
		"conversas", st.Conversations, "mensagens", st.Messages)
	return nil
}

func writePage(path, body string, refresh bool) error {
	meta := ""
	if refresh {
		meta = `<meta http-equiv="refresh" content="3">`
	}
	doc := fmt.Sprintf(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8">%s<title>Parear WhatsApp</title>
<style>body{font-family:system-ui;background:#f6f6f4;color:#1d1d1b;display:flex;flex-direction:column;align-items:center;justify-content:center;min-height:100vh;margin:0;text-align:center;padding:16px}
img{background:#fff;padding:16px;border-radius:12px}.muted{color:#777}.ok{font-size:1.4rem;color:#1a7f37}.code{font:700 2.4rem ui-monospace,monospace;letter-spacing:.2em}</style>
</head><body>%s</body></html>`, meta, body)
	return os.WriteFile(path, []byte(doc), 0o600)
}

// openStore abre o histórico e a sessão sem conectar ao WhatsApp.
func openStore(ctx context.Context) (*whatsapp.MessageStore, func(string) string, func(), error) {
	mgr, err := whatsapp.NewSessionManager()
	if err != nil {
		return nil, nil, nil, err
	}
	dev, err := mgr.Container.GetFirstDevice(ctx)
	if err != nil {
		mgr.Container.Close()
		return nil, nil, nil, fmt.Errorf("carregar dispositivo: %w", err)
	}
	ms, err := whatsapp.OpenMessageStore(ctx, filepath.Join(mgr.ConfigDir, whatsapp.MessagesDBFile))
	if err != nil {
		mgr.Container.Close()
		return nil, nil, nil, err
	}
	cache := map[string]string{}
	resolve := func(jid string) string {
		if n, ok := cache[jid]; ok {
			return n
		}
		n := whatsapp.ResolveName(ctx, dev, jid)
		cache[jid] = n
		return n
	}
	closeAll := func() {
		ms.Close()
		mgr.Container.Close()
	}
	return ms, resolve, closeAll, nil
}

func runChats(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("chats", flag.ExitOnError)
	archived := fs.Bool("archived", false, "apenas arquivados")
	groups := fs.Bool("groups", false, "apenas grupos")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	ms, _, closeAll, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer closeAll()

	chats, err := ms.ListChats(ctx, whatsapp.ChatFilter{ArchivedOnly: *archived, GroupsOnly: *groups})
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ARQ\tGRUPO\tMSGS\tMINHAS\tÚLTIMA\tNOME\tJID")
	for _, c := range chats {
		last := ""
		if !c.LastAt.IsZero() {
			last = c.LastAt.Format(time.DateOnly)
		}
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\t%s\t%s\n", yn(c.Archived), yn(c.IsGroup), c.Messages, c.FromMe, last, c.Name, c.JID)
	}
	fmt.Fprintf(tw, "\n%d chats\n", len(chats))
	return tw.Flush()
}

type exportEntry struct {
	JID          string `json:"jid"`
	Name         string `json:"name"`
	File         string `json:"file"`
	Topic        string `json:"topic,omitempty"`
	Archived     bool   `json:"archived"`
	IsGroup      bool   `json:"is_group"`
	Participants int    `json:"participants,omitempty"`
	Messages     int    `json:"messages"`
	FromMe       int    `json:"from_me"`
	FirstAt      string `json:"first_at,omitempty"`
	LastAt       string `json:"last_at,omitempty"`
	Bytes        int    `json:"bytes"`
}

func runExport(ctx context.Context, argv []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	archived := fs.Bool("archived", false, "apenas arquivados")
	groups := fs.Bool("groups", false, "apenas grupos")
	out := fs.String("out", "data/whatsapp/export", "pasta de saída")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o700); err != nil {
		return fmt.Errorf("criar pasta de exportação: %w", err)
	}
	ms, resolve, closeAll, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer closeAll()

	chats, err := ms.ListChats(ctx, whatsapp.ChatFilter{ArchivedOnly: *archived, GroupsOnly: *groups})
	if err != nil {
		return err
	}

	var index []exportEntry
	used := map[string]bool{}
	for _, c := range chats {
		if c.Messages == 0 {
			continue
		}
		msgs, err := ms.Messages(ctx, c.JID, 0)
		if err != nil {
			return err
		}
		name := c.Name
		if name == "" {
			name = resolve(c.JID)
		}
		file := slugify(name)
		if file == "" || used[file] {
			file = strings.Trim(file+"-"+strings.Split(c.JID, "@")[0], "-")
		}
		used[file] = true
		file += ".md"

		content := renderChat(c, name, msgs, resolve)
		if err := os.WriteFile(filepath.Join(*out, file), []byte(content), 0o600); err != nil {
			return fmt.Errorf("gravar %s: %w", file, err)
		}
		index = append(index, exportEntry{
			JID: c.JID, Name: name, File: file, Topic: c.Topic, Archived: c.Archived, IsGroup: c.IsGroup,
			Participants: c.Participants, Messages: c.Messages, FromMe: c.FromMe,
			FirstAt: c.FirstAt.Format(time.DateOnly), LastAt: c.LastAt.Format(time.DateOnly), Bytes: len(content),
		})
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar índice: %w", err)
	}
	if err := os.WriteFile(filepath.Join(*out, "index.json"), data, 0o600); err != nil {
		return fmt.Errorf("gravar índice: %w", err)
	}
	slog.Info("exportação concluída", "chats", len(index), "pasta", *out)
	return nil
}

func renderChat(c whatsapp.ChatSummary, name string, msgs []whatsapp.Message, resolve func(string) string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	fmt.Fprintf(&b, "- jid: %s\n- grupo: %s\n- arquivado: %s\n", c.JID, yn(c.IsGroup), yn(c.Archived))
	if c.Participants > 0 {
		fmt.Fprintf(&b, "- participantes: %d\n", c.Participants)
	}
	fmt.Fprintf(&b, "- mensagens: %d (minhas: %d)\n- período: %s → %s\n",
		c.Messages, c.FromMe, c.FirstAt.Format(time.DateOnly), c.LastAt.Format(time.DateOnly))
	if c.Topic != "" {
		fmt.Fprintf(&b, "- descrição: %s\n", strings.ReplaceAll(c.Topic, "\n", " "))
	}
	b.WriteString("\n---\n\n")

	byID := make(map[string]whatsapp.Message, len(msgs))
	for _, m := range msgs {
		byID[m.ID] = m
	}
	who := func(m whatsapp.Message) string {
		if m.FromMe {
			return "EU"
		}
		if n := resolve(m.SenderJID); n != "" && !strings.HasPrefix(n, "+") {
			return n
		}
		if m.SenderName != "" {
			return m.SenderName
		}
		if n := resolve(m.SenderAlt); n != "" {
			return n
		}
		return resolve(m.SenderJID)
	}

	for _, m := range msgs {
		line := renderContent(m)
		if q, ok := byID[m.QuotedID]; ok && m.QuotedID != "" {
			line = fmt.Sprintf("↪「%s: %s」 %s", who(q), truncate(oneLine(renderContent(q)), 80), line)
		}
		line = strings.ReplaceAll(line, "\n", "\n    ")
		fmt.Fprintf(&b, "[%s] %s: %s\n", m.Timestamp.Format("2006-01-02 15:04"), who(m), line)
	}
	return b.String()
}

func renderContent(m whatsapp.Message) string {
	switch m.Kind {
	case "text":
		return m.Text
	case "audio":
		if m.Transcript != "" {
			return "[áudio " + m.Text + "] «" + m.Transcript + "»"
		}
		return "[áudio " + m.Text + "]"
	case "image":
		return strings.TrimSpace("[imagem] " + m.Text)
	case "video":
		return strings.TrimSpace("[vídeo] " + m.Text)
	case "document":
		return "[documento] " + m.Text
	case "sticker":
		return "[figurinha]"
	case "contact":
		return "[contato] " + m.Text
	case "location":
		return "[localização] " + m.Text
	case "poll":
		return "[enquete] " + m.Text
	}
	return "[" + m.Kind + "] " + m.Text
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if len(out) > 60 {
		out = strings.TrimRight(out[:60], "-")
	}
	return out
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func yn(v bool) string {
	if v {
		return "sim"
	}
	return "não"
}
