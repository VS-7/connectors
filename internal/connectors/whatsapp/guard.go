package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

// Ações com cota. Tudo que o daemon pede ao WhatsApp (ou ao celular) passa pela Guard.
const (
	ActionSend           = "send_message"    // mensagem enviada pelo operador
	ActionMediaRecent    = "media_download"  // download de mídia recebida nas últimas 24h
	ActionMediaBacklog   = "media_backlog"   // download de mídia antiga
	ActionMediaRetry     = "media_retry"     // pedir ao celular para reenviar mídia expirada
	ActionHistoryRequest = "history_request" // histórico sob demanda (peer message ao celular)
	ActionGroupRefresh   = "group_refresh"   // listar grupos no servidor
)

// ActionPolicy limita uma ação: intervalo mínimo com variação aleatória (sem cadência
// fixa de robô), cotas por hora e por dia e, opcionalmente, só em horário diurno.
type ActionPolicy struct {
	MinGap  time.Duration
	Jitter  time.Duration
	PerHour int
	PerDay  int
	Daytime bool
}

// DefaultPolicies é o "modo seguro": ritmo de uma pessoa usando o WhatsApp Web, não de
// um robô. Mídia nova é tratada como cliente normal; o resto anda devagar e de dia.
func DefaultPolicies() map[string]ActionPolicy {
	return map[string]ActionPolicy{
		ActionSend:           {MinGap: 45 * time.Second, Jitter: 30 * time.Second, PerHour: 10, PerDay: 40, Daytime: true},
		ActionMediaRecent:    {MinGap: 3 * time.Second, Jitter: 5 * time.Second, PerHour: 60, PerDay: 400},
		ActionMediaBacklog:   {MinGap: time.Minute, Jitter: time.Minute, PerHour: 20, PerDay: 100, Daytime: true},
		ActionMediaRetry:     {MinGap: 5 * time.Minute, Jitter: 5 * time.Minute, PerHour: 5, PerDay: 20, Daytime: true},
		ActionHistoryRequest: {MinGap: 15 * time.Minute, Jitter: 10 * time.Minute, PerHour: 2, PerDay: 10, Daytime: true},
		ActionGroupRefresh:   {MinGap: 12 * time.Hour, PerHour: 1, PerDay: 2},
	}
}

// Janela diurna (horário local) para ações com Daytime.
const (
	dayStartHour = 9
	dayEndHour   = 21
)

// ErrBudget indica que a ação estourou a cota; RetryAfter diz quando volta a ser permitida.
type ErrBudget struct {
	Action     string
	RetryAfter time.Time
}

func (e *ErrBudget) Error() string {
	return fmt.Sprintf("limite de segurança para %s: tente após %s", e.Action, e.RetryAfter.Format("02/01 15:04"))
}

// Guard aplica as políticas e registra cada ação no banco (wa_actions), de modo que as
// cotas valem entre reinícios e entre processos.
type Guard struct {
	store    *MessageStore
	policies map[string]ActionPolicy
	now      func() time.Time
	mu       sync.Mutex
	// gaps guarda o intervalo sorteado para a próxima execução de cada ação.
	gaps map[string]time.Duration
}

// NewGuard cria a guarda sobre o banco de histórico.
func NewGuard(store *MessageStore, policies map[string]ActionPolicy) *Guard {
	return &Guard{store: store, policies: policies, now: time.Now, gaps: map[string]time.Duration{}}
}

// Wait bloqueia até a ação ser permitida e a registra. Usado pelos workers de fundo.
func (g *Guard) Wait(ctx context.Context, action string) error {
	for {
		next, err := g.reserve(ctx, action)
		if err != nil || next.IsZero() {
			return err
		}
		if !sleepCtx(ctx, next.Sub(g.now())) {
			return ctx.Err()
		}
	}
}

// Reserve registra a ação se ela for permitida em até maxWait (esperando esse tempo);
// senão devolve *ErrBudget. Usado em chamadas interativas, como o envio de mensagem.
func (g *Guard) Reserve(ctx context.Context, action string, maxWait time.Duration) error {
	next, err := g.reserve(ctx, action)
	if err != nil || next.IsZero() {
		return err
	}
	wait := next.Sub(g.now())
	if wait > maxWait {
		return &ErrBudget{Action: action, RetryAfter: next}
	}
	if !sleepCtx(ctx, wait) {
		return ctx.Err()
	}
	return g.Reserve(ctx, action, maxWait-wait)
}

// reserve registra a ação e devolve zero se permitida agora; senão devolve o próximo
// instante permitido, sem registrar.
func (g *Guard) reserve(ctx context.Context, action string) (time.Time, error) {
	p, ok := g.policies[action]
	if !ok {
		return time.Time{}, fmt.Errorf("ação sem política: %s", action)
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	gap, ok := g.gaps[action]
	if !ok {
		gap = p.MinGap + jitter(p.Jitter)
		g.gaps[action] = gap
	}
	last, hourCount, hourOldest, dayCount, dayOldest, err := g.store.actionWindow(ctx, action, now)
	if err != nil {
		return time.Time{}, err
	}

	next := now
	if !last.IsZero() && last.Add(gap).After(next) {
		next = last.Add(gap)
	}
	if p.PerHour > 0 && hourCount >= p.PerHour {
		next = later(next, hourOldest.Add(time.Hour))
	}
	if p.PerDay > 0 && dayCount >= p.PerDay {
		next = later(next, dayOldest.Add(24*time.Hour))
	}
	if p.Daytime {
		next = daytime(next)
	}
	if next.After(now) {
		return next, nil
	}
	if err := g.store.recordAction(ctx, action, now); err != nil {
		return time.Time{}, err
	}
	delete(g.gaps, action)
	return time.Time{}, nil
}

// Usage devolve quantas vezes cada ação rodou nas últimas 24h.
func (g *Guard) Usage(ctx context.Context) (map[string]int, error) {
	out := map[string]int{}
	for action := range g.policies {
		_, _, _, day, _, err := g.store.actionWindow(ctx, action, g.now())
		if err != nil {
			return nil, err
		}
		out[action] = day
	}
	return out, nil
}

// IsBudget informa se err é estouro de cota.
func IsBudget(err error) bool {
	var b *ErrBudget
	return errors.As(err, &b)
}

func jitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return rand.N(max)
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// daytime empurra t para dentro da janela diurna local, com alguns minutos aleatórios
// para não acordar sempre no mesmo segundo.
func daytime(t time.Time) time.Time {
	start := time.Date(t.Year(), t.Month(), t.Day(), dayStartHour, 0, 0, 0, t.Location())
	end := time.Date(t.Year(), t.Month(), t.Day(), dayEndHour, 0, 0, 0, t.Location())
	switch {
	case t.Before(start):
		return start.Add(jitter(20 * time.Minute))
	case !t.Before(end):
		return start.AddDate(0, 0, 1).Add(jitter(20 * time.Minute))
	}
	return t
}
