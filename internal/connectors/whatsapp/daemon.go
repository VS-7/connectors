package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// ErrLoggedOut indica que o aparelho foi desconectado pelo celular; é preciso parear de novo.
var ErrLoggedOut = errors.New("aparelho desconectado pelo celular; rode omni-wa sync para parear")

// DaemonConfig controla o daemon. O ritmo de tudo que vai ao WhatsApp é da Guard.
type DaemonConfig struct {
	// NewTranscriber monta o transcritor; é chamado de novo a cada 5 min enquanto falhar
	// (ex.: modelo ainda baixando). nil desliga a transcrição.
	NewTranscriber func() (*Transcriber, error)
	// Backlog liga o trabalho sobre mensagens antigas (chaves via histórico sob demanda,
	// download de áudios antigos e media retry). Mídia nova é sempre baixada.
	Backlog bool
	// ControlSocket liga o socket local usado pelo omni-mcp para enviar mensagens.
	ControlSocket bool
}

// recentWindow separa mídia "ao vivo" (tratada como cliente normal) de pendência antiga.
const recentWindow = 24 * time.Hour

// RunDaemon mantém a conexão aberta (mensagens novas e arquivamento são gravados pelo
// handleEvent) e processa a fila de áudio em segundo plano, sempre dentro da Guard.
func (c *Connector) RunDaemon(ctx context.Context, cfg DaemonConfig) error {
	if c.readOnly {
		return ErrSessionBusy
	}
	if !c.waClient.WaitForConnection(time.Minute) {
		return errors.New("whatsapp não conectou em 1 minuto")
	}
	slog.Info("daemon whatsapp conectado", "backlog", cfg.Backlog, "operador", cfg.ControlSocket)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	start := func(fn func(context.Context, DaemonConfig)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(ctx, cfg)
		}()
	}
	start(c.downloadLoop)
	start(c.refreshLoop)
	if cfg.Backlog {
		start(c.backfillLoop)
		start(c.retryLoop)
	}
	if cfg.NewTranscriber != nil {
		start(c.transcribeLoop)
	}
	if cfg.ControlSocket {
		start(func(ctx context.Context, _ DaemonConfig) {
			if err := c.ServeControl(ctx); err != nil {
				slog.Error("socket do operador parou", "error", err)
			}
		})
	}

	var err error
	select {
	case <-ctx.Done():
	case <-c.loggedOut:
		err = ErrLoggedOut
	}
	cancel()
	wg.Wait()
	return err
}

// sleepCtx espera d ou até ctx terminar; devolve false se ctx terminou.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// waitConnected espera a reconexão automática do whatsmeow.
func (c *Connector) waitConnected(ctx context.Context) bool {
	for !c.waClient.IsConnected() {
		if !sleepCtx(ctx, 10*time.Second) {
			return false
		}
	}
	return true
}

// waitBudget dorme até a cota liberar, acordando antes se chegar mídia nova.
func (c *Connector) waitBudget(ctx context.Context, err error) bool {
	var b *ErrBudget
	if !errors.As(err, &b) {
		return sleepCtx(ctx, time.Minute)
	}
	select {
	case <-ctx.Done():
		return false
	case <-c.mediaWake:
	case <-time.After(time.Until(b.RetryAfter)):
	}
	return true
}

func (c *Connector) backfillLoop(ctx context.Context, _ DaemonConfig) {
	for {
		plan, err := c.msgStore.BackfillPlan(ctx, 50)
		if err != nil {
			slog.Warn("falha ao planejar chaves de áudio", "error", err)
		}
		for i, req := range plan {
			if err := c.guard.Wait(ctx, ActionHistoryRequest); err != nil || !c.waitConnected(ctx) {
				return
			}
			c.backfillOne(ctx, req)
			slog.Info("chaves de áudio (histórico sob demanda)", "pedido", i+1, "de", len(plan))
		}
		if !sleepCtx(ctx, 6*time.Hour) {
			return
		}
	}
}

func (c *Connector) backfillOne(ctx context.Context, req BackfillRequest) {
	drain(c.onDemand)
	if err := c.msgStore.MarkKeysRequested(ctx, req); err != nil {
		slog.Warn("falha ao marcar pedido", "error", err)
	}
	if err := c.RequestAudioKeys(ctx, req); err != nil {
		slog.Warn("pedido sob demanda falhou", "chat", req.ChatJID, "error", err)
		return
	}
	c.awaitOnDemand(ctx, req.ChatJID, 45*time.Second)
	got, err := c.msgStore.ResolveBackfill(ctx, req)
	if err != nil {
		slog.Warn("falha ao registrar resposta", "error", err)
	}
	if got > 0 {
		wake(c.mediaWake)
	}
}

// awaitOnDemand espera uma resposta ON_DEMAND do chat pedido (ou o timeout).
func (c *Connector) awaitOnDemand(ctx context.Context, chat string, timeout time.Duration) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case got := <-c.onDemand:
			if got == chat {
				// Respostas grandes podem vir em mais de um lote; dá um respiro para o próximo.
				sleepCtx(ctx, 2*time.Second)
				return
			}
		}
	}
}

// downloadLoop baixa um áudio por vez, sempre o mais novo primeiro: mídia recém-chegada
// passa na frente das pendências antigas, que só andam dentro da cota de backlog.
func (c *Connector) downloadLoop(ctx context.Context, cfg DaemonConfig) {
	for {
		batch, err := c.msgStore.PendingAudioDownloads(ctx, 1)
		if err != nil {
			slog.Warn("falha ao ler fila de download", "error", err)
		}
		if len(batch) == 0 || (!cfg.Backlog && time.Since(batch[0].Timestamp) > recentWindow) {
			select {
			case <-ctx.Done():
				return
			case <-c.mediaWake:
			case <-time.After(10 * time.Minute):
			}
			continue
		}
		m := batch[0]
		action := ActionMediaBacklog
		if time.Since(m.Timestamp) <= recentWindow {
			action = ActionMediaRecent
		}
		if err := c.guard.Reserve(ctx, action, 0); err != nil {
			if !c.waitBudget(ctx, err) {
				return
			}
			continue
		}
		if !c.waitConnected(ctx) {
			return
		}
		if err := c.DownloadAudio(ctx, m); err != nil {
			slog.Warn("download de áudio falhou", "id", m.ID, "error", err)
			if err := c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaFailed); err != nil && !isNotFound(err) {
				slog.Warn("falha ao marcar áudio", "error", err)
			}
		}
	}
}

// retryLoop pede ao celular, devagar, o reenvio de mídias expiradas no servidor.
func (c *Connector) retryLoop(ctx context.Context, _ DaemonConfig) {
	for {
		batch, err := c.msgStore.PendingRetries(ctx, 1)
		if err != nil {
			slog.Warn("falha ao ler fila de reenvio", "error", err)
		}
		if len(batch) == 0 {
			if !sleepCtx(ctx, 30*time.Minute) {
				return
			}
			continue
		}
		if err := c.guard.Wait(ctx, ActionMediaRetry); err != nil || !c.waitConnected(ctx) {
			return
		}
		if err := c.RequestMediaRetry(ctx, batch[0]); err != nil {
			slog.Warn("pedido de reenvio falhou", "id", batch[0].ID, "error", err)
		}
	}
}

func (c *Connector) transcribeLoop(ctx context.Context, cfg DaemonConfig) {
	t, err := cfg.NewTranscriber()
	for warned := false; err != nil; t, err = cfg.NewTranscriber() {
		if !warned {
			slog.Warn("transcrição aguardando dependências; nova tentativa a cada 5 min", "motivo", err)
			warned = true
		}
		if !sleepCtx(ctx, 5*time.Minute) {
			return
		}
	}
	slog.Info("transcrição local ativada", "modelo", t.Model)
	for {
		n, err := TranscribePending(ctx, c.msgStore, t, 10)
		if err != nil {
			slog.Warn("falha na fila de transcrição", "error", err)
		}
		if n > 0 {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-c.transcribeWake:
		case <-time.After(time.Minute):
		}
	}
}

// TranscribePending transcreve até limit áudios baixados e devolve quantos processou.
// É local (whisper.cpp): não fala com o WhatsApp.
func TranscribePending(ctx context.Context, store *MessageStore, t *Transcriber, limit int) (int, error) {
	batch, err := store.PendingTranscriptions(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, m := range batch {
		text, err := t.Transcribe(ctx, m.MediaPath)
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if err != nil {
			slog.Warn("transcrição falhou", "id", m.ID, "error", err)
			if err := store.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaFailed); err != nil {
				return 0, err
			}
			continue
		}
		if err := store.SetTranscript(ctx, m.ChatJID, m.ID, text); err != nil {
			return 0, err
		}
	}
	return len(batch), nil
}

func (c *Connector) refreshLoop(ctx context.Context, _ DaemonConfig) {
	for {
		if c.waitConnected(ctx) {
			if _, err := c.RefreshGroups(ctx); err != nil && !IsBudget(err) {
				slog.Warn("falha ao atualizar grupos", "error", err)
			}
			if _, err := c.ReconcileArchived(ctx); err != nil {
				slog.Warn("falha ao reconciliar arquivados", "error", err)
			}
		}
		if !sleepCtx(ctx, 12*time.Hour+jitter(time.Hour)) {
			return
		}
	}
}

func drain(ch chan string) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
