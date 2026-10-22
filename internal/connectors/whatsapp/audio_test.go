package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestOpenMessageStore_MigraSchemaAntigo(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "antigo.db")
	db, err := sql.Open("sqlite3", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(historySchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO messages (chat_jid, id, timestamp, kind, text) VALUES ('1@g.us', 'a', 1, 'audio', '12s')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	s, err := OpenMessageStore(ctx, path)
	if err != nil {
		t.Fatalf("migrar: %v", err)
	}
	defer s.Close()
	msgs, err := s.Messages(ctx, "1@g.us", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Text != "12s" || msgs[0].Media != nil || msgs[0].MediaStatus != MediaPending {
		t.Fatalf("mensagem antiga deveria sobreviver à migração sem mídia: %+v", msgs)
	}
}

func TestSaveMessages_ChaveTardiaVoltaParaFilaDeDownload(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	audio := Message{ChatJID: "1@g.us", ID: "a", Timestamp: time.Unix(10, 0), Kind: "audio", Text: "12s"}
	if err := s.SaveMessages(ctx, []Message{audio}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMediaStatus(ctx, "1@g.us", "a", MediaKeysUnavailable); err != nil {
		t.Fatal(err)
	}

	audio.Media = []byte{1, 2, 3}
	if err := s.SaveMessages(ctx, []Message{audio}); err != nil {
		t.Fatal(err)
	}
	pend, err := s.PendingAudioDownloads(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || string(pend[0].Media) != "\x01\x02\x03" {
		t.Fatalf("áudio com chave nova deveria entrar na fila de download: %+v", pend)
	}

	// Reentrega sem mídia (ex.: lote de histórico antigo) não apaga a chave nem o estado.
	audio.Media = nil
	if err := s.SetMediaFile(ctx, "1@g.us", "a", "/tmp/a.ogg"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessages(ctx, []Message{audio}); err != nil {
		t.Fatal(err)
	}
	got, _, err := s.MessageByID(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Media == nil || got.MediaStatus != MediaDownloaded || got.MediaPath != "/tmp/a.ogg" {
		t.Fatalf("reentrega sem mídia não deveria regredir o áudio: %+v", got)
	}
}

func TestPlanChat(t *testing.T) {
	msg := func(id, kind string) Message { return Message{ID: id, Kind: kind} }
	comChave := Message{ID: "x", Kind: "audio", Media: []byte{1}}
	casos := []struct {
		nome     string
		msgs     []Message
		max      int
		querReqs [][]string // ids de áudio por pedido
		querAnc  []string
		querCnt  []int
	}{
		{"áudio único com âncora", []Message{msg("1", "text"), msg("2", "audio"), msg("3", "text")}, 50,
			[][]string{{"2"}}, []string{"3"}, []int{1}},
		{"áudio no fim do chat fica sem âncora", []Message{msg("1", "text"), msg("2", "audio")}, 50, nil, nil, nil},
		{"áudios próximos num pedido só", []Message{msg("1", "audio"), msg("2", "text"), msg("3", "audio"), msg("4", "text")}, 50,
			[][]string{{"3", "1"}}, []string{"4"}, []int{3}},
		{"limite divide em dois pedidos", []Message{msg("1", "audio"), msg("2", "text"), msg("3", "audio"), msg("4", "text")}, 2,
			[][]string{{"3"}, {"1"}}, []string{"4", "2"}, []int{1, 1}},
		{"áudio que já tem chave é ignorado", []Message{comChave, msg("2", "text")}, 50, nil, nil, nil},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			reqs := planChat("1@g.us", tc.msgs, tc.max)
			if len(reqs) != len(tc.querReqs) {
				t.Fatalf("quer %d pedidos, obteve %d: %+v", len(tc.querReqs), len(reqs), reqs)
			}
			for i, r := range reqs {
				if r.Anchor.ID != tc.querAnc[i] || r.Count != tc.querCnt[i] || len(r.AudioIDs) != len(tc.querReqs[i]) {
					t.Errorf("pedido %d: âncora %s count %d ids %v", i, r.Anchor.ID, r.Count, r.AudioIDs)
				}
			}
		})
	}
}

func TestResolveBackfill(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	msgs := []Message{
		{ChatJID: "1@g.us", ID: "a", Timestamp: time.Unix(1, 0), Kind: "audio", Media: []byte{1}},
		{ChatJID: "1@g.us", ID: "b", Timestamp: time.Unix(2, 0), Kind: "audio"},
	}
	if err := s.SaveMessages(ctx, msgs); err != nil {
		t.Fatal(err)
	}
	got, err := s.ResolveBackfill(ctx, BackfillRequest{ChatJID: "1@g.us", AudioIDs: []string{"a", "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Errorf("quer 1 áudio com chave, obteve %d", got)
	}
	b, _, _ := s.MessageByID(ctx, "b")
	if b.MediaStatus != MediaKeysUnavailable {
		t.Errorf("áudio sem resposta deveria ficar %q, ficou %q", MediaKeysUnavailable, b.MediaStatus)
	}
}

func TestSessionLock(t *testing.T) {
	dir := t.TempDir()
	l, err := AcquireSessionLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireSessionLock(dir); !errors.Is(err, ErrSessionBusy) {
		t.Fatalf("segundo lock deveria falhar com ErrSessionBusy, obteve %v", err)
	}
	if SessionHolder(dir) == "" {
		t.Error("SessionHolder deveria apontar o dono do lock")
	}
	if err := l.Release(); err != nil {
		t.Fatal(err)
	}
	l2, err := AcquireSessionLock(dir)
	if err != nil {
		t.Fatalf("lock liberado deveria poder ser obtido de novo: %v", err)
	}
	l2.Release()
}

func TestCleanTranscript(t *testing.T) {
	casos := []struct{ in, quer string }{
		{" Oi Ana, tudo bem?\n Consegue ver o banner hoje?\n", "Oi Ana, tudo bem? Consegue ver o banner hoje?"},
		{"[BLANK_AUDIO]\n", ""},
		{"(música)\n Bom dia\n", "Bom dia"},
	}
	for _, tc := range casos {
		if got := cleanTranscript(tc.in); got != tc.quer {
			t.Errorf("cleanTranscript(%q) = %q, quer %q", tc.in, got, tc.quer)
		}
	}
}

func TestToMessage_GuardaChavesDoAudio(t *testing.T) {
	chat := types.NewJID("123", types.GroupServer)
	evt := &events.Message{
		Info: types.MessageInfo{MessageSource: types.MessageSource{Chat: chat, Sender: types.NewJID("55", types.DefaultUserServer)},
			ID: "x", Timestamp: time.Unix(5, 0)},
		Message: &waE2E.Message{AudioMessage: &waE2E.AudioMessage{Seconds: proto.Uint32(9), MediaKey: []byte("k"), DirectPath: proto.String("/p")}},
	}
	m, ok := toMessage(evt)
	if !ok || m.Kind != "audio" || m.Media == nil {
		t.Fatalf("áudio deveria guardar a mídia: %+v", m)
	}
	var audio waE2E.AudioMessage
	if err := proto.Unmarshal(m.Media, &audio); err != nil || audio.GetDirectPath() != "/p" {
		t.Fatalf("mídia guardada inválida: %v %v", err, audio.GetDirectPath())
	}
}
