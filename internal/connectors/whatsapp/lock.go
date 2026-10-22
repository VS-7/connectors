package whatsapp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ErrSessionBusy indica que outro processo (normalmente o daemon omni-wa) já está
// conectado com esta sessão. Duas conexões simultâneas derrubam uma à outra.
var ErrSessionBusy = errors.New("sessão do WhatsApp em uso por outro processo")

// SessionLock é um flock exclusivo sobre whatsapp.lock no diretório de configuração.
type SessionLock struct {
	f *os.File
}

// AcquireSessionLock tenta obter o lock sem bloquear.
func AcquireSessionLock(configDir string) (*SessionLock, error) {
	path := filepath.Join(configDir, "whatsapp.lock")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("abrir lock da sessão: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			if pid := lockHolder(path); pid != "" {
				return nil, fmt.Errorf("%w (pid %s)", ErrSessionBusy, pid)
			}
			return nil, ErrSessionBusy
		}
		return nil, fmt.Errorf("travar sessão: %w", err)
	}
	// O pid é só informativo para mensagens de erro; falhar ao gravá-lo não impede o lock.
	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0)
	}
	return &SessionLock{f: f}, nil
}

// Release libera o lock.
func (l *SessionLock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	defer l.f.Close()
	return syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
}

// SessionHolder devolve o pid de quem segura o lock, ou "" se ninguém segura.
func SessionHolder(configDir string) string {
	l, err := AcquireSessionLock(configDir)
	if err == nil {
		l.Release()
		return ""
	}
	if pid := lockHolder(filepath.Join(configDir, "whatsapp.lock")); pid != "" {
		return pid
	}
	return "?"
}

// daemonAlive diz se o último dono registrado da sessão ainda está rodando. Só lê o pid
// gravado, sem tentar o lock: segurá-lo mesmo por um instante poderia fazer o daemon,
// subindo naquele momento, achar a sessão ocupada e sair.
func daemonAlive(configDir string) (int, bool) {
	pid, err := strconv.Atoi(lockHolder(filepath.Join(configDir, "whatsapp.lock")))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, syscall.Kill(pid, 0) == nil
}

func lockHolder(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
