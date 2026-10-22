package whatsapp

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func novoStore(t *testing.T) *MessageStore {
	t.Helper()
	s, err := OpenMessageStore(context.Background(), filepath.Join(t.TempDir(), "msgs.db"))
	if err != nil {
		t.Fatalf("abrir store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMessageStore_ArquivadoSobreviveAMetadadosDeGrupo(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	jid := "123@g.us"

	if err := s.UpsertChat(ctx, Chat{JID: jid, Name: "Projeto X", IsGroup: true, Archived: true, UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateGroupMeta(ctx, jid, "Projeto X - Cliente", "descrição", 12); err != nil {
		t.Fatal(err)
	}

	chats, err := s.ListChats(ctx, ChatFilter{ArchivedOnly: true, GroupsOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 {
		t.Fatalf("esperava 1 grupo arquivado, obteve %d", len(chats))
	}
	c := chats[0]
	if c.Name != "Projeto X - Cliente" || c.Topic != "descrição" || c.Participants != 12 || !c.Archived {
		t.Errorf("metadados inesperados: %+v", c.Chat)
	}
}

func TestMessageStore_EdicaoSubstituiTextoEMessagesRespeitaLimite(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	jid := "456@g.us"
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	msgs := []Message{
		{ChatJID: jid, ID: "a", Timestamp: base, Kind: "text", Text: "primeira"},
		{ChatJID: jid, ID: "b", Timestamp: base.Add(time.Minute), Kind: "text", Text: "segunda", FromMe: true},
		{ChatJID: jid, ID: "c", Timestamp: base.Add(2 * time.Minute), Kind: "text", Text: "terceira"},
	}
	if err := s.SaveMessages(ctx, msgs); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessages(ctx, []Message{{ChatJID: jid, ID: "b", Timestamp: base.Add(5 * time.Minute), Kind: "text", Text: "segunda editada"}}); err != nil {
		t.Fatal(err)
	}

	got, err := s.Messages(ctx, jid, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "c" {
		t.Fatalf("esperava [b c] em ordem cronológica, obteve %+v", got)
	}
	if got[0].Text != "segunda editada" || !got[0].Timestamp.Equal(base.Add(time.Minute)) {
		t.Errorf("edição deveria trocar só o texto: %+v", got[0])
	}
}

func TestMessageStore_MessagesSinceFicaComAsMaisRecentesEmOrdem(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	msgs := []Message{
		{ChatJID: "1@g.us", ID: "velha", Timestamp: base.Add(-time.Hour), Kind: "text", Text: "fora da janela"},
		{ChatJID: "1@g.us", ID: "a", Timestamp: base, Kind: "text", Text: "a"},
		{ChatJID: "2@s.whatsapp.net", ID: "b", Timestamp: base.Add(time.Minute), Kind: "text", Text: "b"},
		{ChatJID: "1@g.us", ID: "c", Timestamp: base.Add(2 * time.Minute), Kind: "text", Text: "c"},
	}
	if err := s.SaveMessages(ctx, msgs); err != nil {
		t.Fatal(err)
	}

	got, err := s.MessagesSince(ctx, base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "c" {
		t.Fatalf("esperava [b c] (as 2 mais recentes, em ordem cronológica), obteve %+v", got)
	}

	todas, err := s.MessagesSince(ctx, base, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(todas) != 3 {
		t.Errorf("esperava 3 mensagens desde base, obteve %d", len(todas))
	}
}

func TestMessageStore_SearchMessagesExigeTodosOsTermos(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	msgs := []Message{
		{ChatJID: "1@g.us", ID: "a", Timestamp: base, Kind: "text", Text: "o FRETE da loja subiu"},
		{ChatJID: "1@g.us", ID: "b", Timestamp: base.Add(time.Minute), Kind: "audio", Text: "12s"},
		{ChatJID: "2@g.us", ID: "c", Timestamp: base.Add(2 * time.Minute), Kind: "text", Text: "frete ok"},
		{ChatJID: "2@g.us", ID: "d", Timestamp: base.Add(3 * time.Minute), Kind: "text", Text: "desconto de 100% na loja"},
	}
	if err := s.SaveMessages(ctx, msgs); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTranscript(ctx, "1@g.us", "b", "precisa revisar o frete grátis da loja"); err != nil {
		t.Fatal(err)
	}

	got, err := s.SearchMessages(ctx, "frete loja", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "a" {
		t.Fatalf("esperava [b a] (transcrição incluída, mais recente primeiro), obteve %+v", got)
	}

	got, err = s.SearchMessages(ctx, "frete", "2@g.us", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "c" {
		t.Fatalf("esperava só [c] no chat 2, obteve %+v", got)
	}

	// % e _ são literais, não curingas do LIKE.
	got, err = s.SearchMessages(ctx, "100%", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "d" {
		t.Fatalf("esperava só [d] para 100%%, obteve %+v", got)
	}
	if got, _ := s.SearchMessages(ctx, "_", "", 10); len(got) != 0 {
		t.Errorf("_ não deveria casar com qualquer caractere: %+v", got)
	}

	if _, err := s.SearchMessages(ctx, "   ", "", 10); err == nil {
		t.Error("busca vazia deveria falhar")
	}
}

func TestMessageStore_ChatsELastMessageAt(t *testing.T) {
	ctx := context.Background()
	s := novoStore(t)

	last, err := s.LastMessageAt(ctx)
	if err != nil || !last.IsZero() {
		t.Fatalf("store vazio deveria devolver zero: %v %v", last, err)
	}

	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	if err := s.UpsertChat(ctx, Chat{JID: "1@g.us", Name: "Cliente X", IsGroup: true, Archived: true, UpdatedAt: ts}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessages(ctx, []Message{{ChatJID: "1@g.us", ID: "a", Timestamp: ts, Kind: "text", Text: "oi"}}); err != nil {
		t.Fatal(err)
	}

	chats, err := s.Chats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c := chats["1@g.us"]; c.Name != "Cliente X" || !c.IsGroup || !c.Archived {
		t.Errorf("chat inesperado: %+v", c)
	}
	if last, err := s.LastMessageAt(ctx); err != nil || !last.Equal(ts) {
		t.Errorf("LastMessageAt = %v, %v; esperava %v", last, err, ts)
	}
}

func TestExtractContent(t *testing.T) {
	casos := []struct {
		nome     string
		msg      *waE2E.Message
		querKind string
		querText string
		querOK   bool
	}{
		{"texto simples", &waE2E.Message{Conversation: proto.String("oi")}, "text", "oi", true},
		{"texto estendido", &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("link")}}, "text", "link", true},
		{"áudio", &waE2E.Message{AudioMessage: &waE2E.AudioMessage{Seconds: proto.Uint32(42)}}, "audio", "42s", true},
		{"documento com legenda", &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{FileName: proto.String("escopo.pdf"), Caption: proto.String("v2")}}, "document", "escopo.pdf — v2", true},
		{"reação é ignorada", &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{Text: proto.String("👍")}}, "", "", false},
		{"nil", nil, "", "", false},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			kind, text, _, ok := extractContent(tc.msg)
			if kind != tc.querKind || text != tc.querText || ok != tc.querOK {
				t.Errorf("obteve (%q, %q, %v), quer (%q, %q, %v)", kind, text, ok, tc.querKind, tc.querText, tc.querOK)
			}
		})
	}
}
