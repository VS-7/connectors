package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const controlSocketFile = "omni-wa.sock"

// ServeControl expõe ao usuário local (socket Unix com permissão 0600) o envio de
// mensagens pelo daemon. O omni-mcp em modo leitura encaminha whatsapp_send_message para
// cá, de modo que existe uma única conexão com o WhatsApp e uma única Guard.
func (c *Connector) ServeControl(ctx context.Context) error {
	path := filepath.Join(c.sessionMgr.ConfigDir, controlSocketFile)
	// Socket de um daemon anterior que caiu; quem chega aqui já tem o lock da sessão.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remover socket antigo: %w", err)
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return fmt.Errorf("abrir socket do operador: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return fmt.Errorf("proteger socket do operador: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Recipient string `json:"recipient"`
			Message   string `json:"message"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "json inválido"})
			return
		}
		res, err := c.SendGuarded(r.Context(), req.Recipient, req.Message)
		if err != nil {
			body := map[string]any{"error": err.Error()}
			var b *ErrBudget
			if errors.As(err, &b) {
				body["retry_after"] = b.RetryAfter.Format(time.RFC3339)
			}
			writeJSON(w, http.StatusUnprocessableEntity, body)
			return
		}
		writeJSON(w, http.StatusOK, res)
	})
	mux.HandleFunc("GET /usage", func(w http.ResponseWriter, r *http.Request) {
		usage, err := c.guard.Usage(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, usage)
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// SendViaDaemon encaminha um envio ao daemon pelo socket local.
func SendViaDaemon(ctx context.Context, configDir, recipient, message string) (map[string]any, error) {
	path := filepath.Join(configDir, controlSocketFile)
	client := &http.Client{
		Timeout: 3 * time.Minute, // cobre a espera pela cota (até 90 s) e o "digitando…"
		Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", path)
		}},
	}
	payload, err := json.Marshal(map[string]string{"recipient": recipient, "message": message})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://omni-wa/send", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daemon omni-wa indisponível para envio: %w", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("resposta inválida do daemon: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		msg, _ := out["error"].(string)
		if ra, ok := out["retry_after"].(string); ok {
			msg += " (tente após " + ra + ")"
		}
		return nil, errors.New(msg)
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Erro de escrita aqui significa cliente desconectado; não há o que fazer.
	_ = json.NewEncoder(w).Encode(v)
}
