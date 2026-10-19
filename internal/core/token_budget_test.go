package core_test

import (
	"strings"
	"testing"

	"connectors/internal/core"
)

func TestTruncateText(t *testing.T) {
	short := "Texto curto"
	if res := core.TruncateText(short, 50); res != short {
		t.Errorf("esperado '%s', recebido '%s'", short, res)
	}

	long := strings.Repeat("palavra ", 50)
	res := core.TruncateText(long, 40)
	if !strings.Contains(res, "[... restante truncado para economia de tokens ...]") {
		t.Errorf("esperava indicador de truncamento, recebido: %s", res)
	}
	if len(res) >= len(long) {
		t.Errorf("resultado truncado deveria ser menor que o original")
	}
}

func TestStripHTML(t *testing.T) {
	input := "<html><body><h1>Título</h1><p>Parágrafo com &nbsp;espaço e <b>negrito</b>.</p></body></html>"
	expected := "Título Parágrafo com espaço e negrito."
	res := core.StripHTML(input)
	if res != expected {
		t.Errorf("esperado '%s', recebido '%s'", expected, res)
	}
}
