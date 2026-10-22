package whatsapp

import (
	"context"
	"errors"
	"testing"
	"time"
)

type relogio struct{ t time.Time }

func (r *relogio) now() time.Time        { return r.t }
func (r *relogio) passa(d time.Duration) { r.t = r.t.Add(d) }
func novaGuard(t *testing.T, p ActionPolicy, inicio time.Time) (*Guard, *relogio) {
	t.Helper()
	r := &relogio{t: inicio}
	g := NewGuard(novoStore(t), map[string]ActionPolicy{"x": p})
	g.now = r.now
	return g, r
}

func retryAfter(t *testing.T, err error) time.Time {
	t.Helper()
	var b *ErrBudget
	if !errors.As(err, &b) {
		t.Fatalf("esperava ErrBudget, obteve %v", err)
	}
	return b.RetryAfter
}

var meioDia = time.Date(2026, 10, 22, 12, 0, 0, 0, time.Local)

func TestGuard_IntervaloMinimo(t *testing.T) {
	ctx := context.Background()
	g, r := novaGuard(t, ActionPolicy{MinGap: 10 * time.Second}, meioDia)
	if err := g.Reserve(ctx, "x", 0); err != nil {
		t.Fatalf("primeira execução deveria passar: %v", err)
	}
	if ra := retryAfter(t, g.Reserve(ctx, "x", 0)); !ra.Equal(meioDia.Add(10 * time.Second)) {
		t.Errorf("próxima execução em %v, quer %v", ra, meioDia.Add(10*time.Second))
	}
	r.passa(10 * time.Second)
	if err := g.Reserve(ctx, "x", 0); err != nil {
		t.Errorf("depois do intervalo deveria passar: %v", err)
	}
}

func TestGuard_CotaPorHoraEPorDia(t *testing.T) {
	ctx := context.Background()
	g, r := novaGuard(t, ActionPolicy{PerHour: 2, PerDay: 3}, meioDia)
	for i := 0; i < 2; i++ {
		if err := g.Reserve(ctx, "x", 0); err != nil {
			t.Fatalf("execução %d deveria passar: %v", i+1, err)
		}
		r.passa(time.Minute)
	}
	if ra := retryAfter(t, g.Reserve(ctx, "x", 0)); !ra.Equal(meioDia.Add(time.Hour)) {
		t.Errorf("cota por hora: libera em %v, quer %v", ra, meioDia.Add(time.Hour))
	}
	r.passa(time.Hour)
	if err := g.Reserve(ctx, "x", 0); err != nil {
		t.Fatalf("depois de 1h deveria passar: %v", err)
	}
	r.passa(time.Hour)
	if ra := retryAfter(t, g.Reserve(ctx, "x", 0)); !ra.Equal(meioDia.Add(24 * time.Hour)) {
		t.Errorf("cota diária: libera em %v, quer %v", ra, meioDia.Add(24*time.Hour))
	}
	usage, err := g.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usage["x"] != 3 {
		t.Errorf("uso em 24h = %d, quer 3", usage["x"])
	}
}

func TestGuard_SoDeDia(t *testing.T) {
	ctx := context.Background()
	noite := time.Date(2026, 10, 22, 23, 30, 0, 0, time.Local)
	g, _ := novaGuard(t, ActionPolicy{Daytime: true}, noite)
	ra := retryAfter(t, g.Reserve(ctx, "x", 0))
	inicio := time.Date(2026, 10, 23, dayStartHour, 0, 0, 0, time.Local)
	if ra.Before(inicio) || ra.After(inicio.Add(20*time.Minute)) {
		t.Errorf("de noite deveria esperar a manhã seguinte (~%v), obteve %v", inicio, ra)
	}
}

func TestGuard_IntervaloComVariacao(t *testing.T) {
	ctx := context.Background()
	g, _ := novaGuard(t, ActionPolicy{MinGap: time.Minute, Jitter: time.Minute}, meioDia)
	if err := g.Reserve(ctx, "x", 0); err != nil {
		t.Fatal(err)
	}
	ra := retryAfter(t, g.Reserve(ctx, "x", 0))
	if gap := ra.Sub(meioDia); gap < time.Minute || gap >= 2*time.Minute {
		t.Errorf("intervalo %v fora de [1m, 2m)", gap)
	}
	// O intervalo sorteado se mantém até a ação rodar (não muda a cada consulta).
	if ra2 := retryAfter(t, g.Reserve(ctx, "x", 0)); !ra2.Equal(ra) {
		t.Errorf("intervalo mudou entre consultas: %v → %v", ra, ra2)
	}
}

func TestOperador_ConversaPreviaERepeticao(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	msgs := []Message{
		{ChatJID: "111@lid", ID: "a", Timestamp: meioDia, Kind: "text", Text: "oi, tudo bem?"},
		{ChatJID: "222@s.whatsapp.net", ID: "b", Timestamp: meioDia, Kind: "text", Text: "bom dia", FromMe: true},
	}
	if err := s.SaveMessages(ctx, msgs); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nome string
		jids []string
		quer bool
	}{
		{"pessoa escreveu (via alias LID)", []string{"5533@s.whatsapp.net", "111@lid"}, true},
		{"só eu escrevi", []string{"222@s.whatsapp.net"}, false},
		{"chat desconhecido", []string{"999@s.whatsapp.net"}, false},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			got, err := s.HasInbound(ctx, tc.jids)
			if err != nil || got != tc.quer {
				t.Errorf("HasInbound(%v) = %v, %v; quer %v", tc.jids, got, err, tc.quer)
			}
		})
	}

	h := textHash("Seu site novo está no ar, segue o link para validação")
	if err := s.RecordOperatorSend(ctx, "111@lid", h, meioDia); err != nil {
		t.Fatal(err)
	}
	if dup, _ := s.SentElsewhere(ctx, h, "333@lid", meioDia.Add(-time.Hour)); !dup {
		t.Error("mesmo texto para outro chat na última hora deveria contar como repetição")
	}
	if dup, _ := s.SentElsewhere(ctx, h, "111@lid", meioDia.Add(-time.Hour)); dup {
		t.Error("repetir no mesmo chat não é disparo em massa")
	}
	if textHash("  Seu site NOVO está no ar,  segue o link para validação ") != h {
		t.Error("hash deveria ignorar caixa e espaços extras")
	}
}
