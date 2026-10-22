package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	signallog "go.mau.fi/libsignal/logger"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// slogLogger adapta o logger do whatsmeow ao slog do processo, que escreve no stderr
// (o stdout do omni-mcp é reservado ao JSON-RPC).
type slogLogger struct {
	l   *slog.Logger
	min slog.Level
}

// newWALogger devolve um logger do whatsmeow no nível pedido (DEBUG, INFO, WARN, ERROR)
// ou um logger mudo quando level é vazio.
func newWALogger(level string) waLog.Logger {
	var min slog.Level
	if level == "" || min.UnmarshalText([]byte(strings.ToUpper(level))) != nil {
		return waLog.Noop
	}
	return slogLogger{l: slog.Default().With("module", "whatsmeow"), min: min}
}

func (s slogLogger) log(level slog.Level, msg string, args ...any) {
	if level < s.min {
		return
	}
	s.l.Log(context.Background(), level, fmt.Sprintf(msg, args...))
}

func (s slogLogger) Debugf(msg string, args ...any) { s.log(slog.LevelDebug, msg, args...) }
func (s slogLogger) Infof(msg string, args ...any)  { s.log(slog.LevelInfo, msg, args...) }
func (s slogLogger) Warnf(msg string, args ...any)  { s.log(slog.LevelWarn, msg, args...) }
func (s slogLogger) Errorf(msg string, args ...any) { s.log(slog.LevelError, msg, args...) }

func (s slogLogger) Sub(module string) waLog.Logger {
	return slogLogger{l: s.l.With("sub", module), min: s.min}
}

// signalLogger tira os logs do libsignal do stdout (o logger padrão usa fmt.Println,
// o que corromperia o JSON-RPC do omni-mcp). Debug é descartado: inclui trechos cifrados.
type signalLogger struct{}

func (signalLogger) Debug(caller, msg string) {}
func (signalLogger) Info(caller, msg string) {
	slog.Info(msg, "module", "libsignal", "caller", caller)
}
func (signalLogger) Warning(caller, msg string) {
	slog.Warn(msg, "module", "libsignal", "caller", caller)
}
func (signalLogger) Error(caller, msg string) {
	slog.Error(msg, "module", "libsignal", "caller", caller)
}
func (signalLogger) Configure(string) {}

var setupSignalLogger = sync.OnceFunc(func() {
	var l signallog.Loggable = signalLogger{}
	signallog.Setup(&l)
})
