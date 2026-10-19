package core

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	htmlTagRegex   = regexp.MustCompile(`<[^>]*>`)
	extraSpaceRegex = regexp.MustCompile(`\s+`)
)

// TruncateText trunca um texto que ultrapassa maxChars preservando palavras e adicionando indicador.
func TruncateText(text string, maxChars int) string {
	if maxChars <= 0 || len(text) <= maxChars {
		return text
	}

	truncated := text[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxChars/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "\n\n[... restante truncado para economia de tokens ...]"
}

// StripHTML remove tags HTML e normaliza espaços repetidos para produzir texto puro compacto.
func StripHTML(input string) string {
	clean := htmlTagRegex.ReplaceAllString(input, " ")
	clean = strings.ReplaceAll(clean, "&nbsp;", " ")
	clean = strings.ReplaceAll(clean, "&amp;", "&")
	clean = strings.ReplaceAll(clean, "&lt;", "<")
	clean = strings.ReplaceAll(clean, "&gt;", ">")
	clean = strings.ReplaceAll(clean, "&quot;", "\"")
	clean = extraSpaceRegex.ReplaceAllString(clean, " ")
	clean = strings.ReplaceAll(clean, " .", ".")
	clean = strings.ReplaceAll(clean, " ,", ",")
	clean = strings.ReplaceAll(clean, " !", "!")
	clean = strings.ReplaceAll(clean, " ?", "?")
	clean = strings.ReplaceAll(clean, " ;", ";")
	clean = strings.ReplaceAll(clean, " :", ":")
	return strings.TrimSpace(clean)

}

// CompactJSON serializa uma estrutura qualquer em JSON compacto sem espaços redundantes.
func CompactJSON(v any) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
