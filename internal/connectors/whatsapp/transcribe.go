package whatsapp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultWhisperModel é o modelo esperado em <configDir>/models quando WHISPER_MODEL não é definido.
const DefaultWhisperModel = "ggml-large-v3-turbo-q5_0.bin"

// Transcriber transcreve áudios localmente com whisper.cpp (whisper-cli), convertendo
// o OGG/Opus do WhatsApp para WAV 16 kHz com ffmpeg. Nada sai da máquina.
type Transcriber struct {
	WhisperBin string
	FFmpegBin  string
	Model      string
	Language   string
}

// NewTranscriberFromEnv monta o transcritor a partir de WHISPER_BIN, WHISPER_MODEL,
// WHISPER_LANG e FFMPEG_BIN. Devolve erro se o binário ou o modelo não existirem.
func NewTranscriberFromEnv(configDir string) (*Transcriber, error) {
	t := &Transcriber{
		WhisperBin: envOr("WHISPER_BIN", "whisper-cli"),
		FFmpegBin:  envOr("FFMPEG_BIN", "ffmpeg"),
		Model:      envOr("WHISPER_MODEL", filepath.Join(configDir, "models", DefaultWhisperModel)),
		Language:   envOr("WHISPER_LANG", "pt"),
	}
	for _, bin := range []*string{&t.WhisperBin, &t.FFmpegBin} {
		path, err := exec.LookPath(*bin)
		if err != nil {
			return nil, fmt.Errorf("transcrição indisponível: %s não encontrado no PATH", *bin)
		}
		*bin = path
	}
	if _, err := os.Stat(t.Model); err != nil {
		return nil, fmt.Errorf("transcrição indisponível: modelo %s não encontrado", t.Model)
	}
	return t, nil
}

// Transcribe devolve o texto falado no arquivo de áudio.
func (t *Transcriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	tmp, err := os.MkdirTemp("", "omni-wa-whisper-")
	if err != nil {
		return "", fmt.Errorf("criar pasta temporária: %w", err)
	}
	defer os.RemoveAll(tmp)

	wav := filepath.Join(tmp, "audio.wav")
	conv := exec.CommandContext(ctx, t.FFmpegBin, "-nostdin", "-loglevel", "error", "-y",
		"-i", audioPath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav)
	if out, err := conv.CombinedOutput(); err != nil {
		return "", fmt.Errorf("converter áudio: %w: %s", err, strings.TrimSpace(string(out)))
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, t.WhisperBin, "-m", t.Model, "-l", t.Language, "-nt", "-np", "-f", wav)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper: %w: %s", err, lastLine(stderr.String()))
	}
	return cleanTranscript(stdout.String()), nil
}

// cleanTranscript junta as linhas do whisper e remove marcações de silêncio/ruído.
func cleanTranscript(raw string) string {
	var parts []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "", strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"),
			strings.HasPrefix(line, "(") && strings.HasSuffix(line, ")"):
			continue
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " ")
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
