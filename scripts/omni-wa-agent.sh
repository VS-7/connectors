#!/usr/bin/env bash
# Instala o `omni-wa daemon` como LaunchAgent do macOS: sobe no login, reinicia se cair
# e mantém o WhatsApp sincronizado (mensagens novas, arquivados, áudios e transcrição).
#
#   scripts/omni-wa-agent.sh install    compila, instala em ~/.local/bin e carrega o agente
#   scripts/omni-wa-agent.sh uninstall  descarrega e remove o agente (necessário antes de `omni-wa sync`)
#   scripts/omni-wa-agent.sh status     estado do agente e andamento dos áudios
#   scripts/omni-wa-agent.sh logs       acompanha o log
#   scripts/omni-wa-agent.sh claude     dá o WhatsApp a toda sessão do Claude Code: instala o omni-mcp,
#                                       registra o servidor MCP `omni` (escopo user, só leitura — quem
#                                       conecta é o daemon) e copia a skill whatsapp-contexto
#
# Opções (variáveis de ambiente no momento do install):
#   OMNI_WA_BACKLOG=on   processa áudios antigos devagar (histórico sob demanda e media retry).
#                        Padrão: desligado — o daemon age como um cliente normal.
#   OMNI_WA_SEND=off     desliga o envio de mensagens pelo operador.
set -euo pipefail

LABEL="dev.omni-connectors.omni-wa"
BIN="$HOME/.local/bin/omni-wa"
MCP_BIN="$HOME/.local/bin/omni-mcp"
SKILL="whatsapp-contexto"
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"
LOG="$HOME/Library/Logs/omni-wa.log"
REPO="$(cd "$(dirname "$0")/.." && pwd)"
DOMAIN="gui/$(id -u)"

case "${1:-}" in
install)
  mkdir -p "$(dirname "$BIN")" "$(dirname "$PLIST")" "$(dirname "$LOG")"
  (cd "$REPO" && go build -o "$BIN" ./cmd/omni-wa)
  cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$LABEL</string>
  <key>ProgramArguments</key>
  <array><string>$BIN</string><string>daemon</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>ThrottleInterval</key><integer>60</integer>
  <key>ProcessType</key><string>Background</string>
  <key>StandardOutPath</key><string>$LOG</string>
  <key>StandardErrorPath</key><string>$LOG</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key><string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
    <key>OMNI_WA_BACKLOG</key><string>${OMNI_WA_BACKLOG:-off}</string>
    <key>OMNI_WA_SEND</key><string>${OMNI_WA_SEND:-on}</string>
  </dict>
</dict>
</plist>
EOF
  launchctl bootout "$DOMAIN/$LABEL" 2>/dev/null || true
  launchctl bootstrap "$DOMAIN" "$PLIST"
  echo "agente $LABEL instalado — log em $LOG"
  ;;
uninstall)
  launchctl bootout "$DOMAIN/$LABEL" 2>/dev/null || true
  rm -f "$PLIST"
  echo "agente $LABEL removido"
  ;;
status)
  launchctl print "$DOMAIN/$LABEL" 2>/dev/null | grep -E "^\s+(state|pid|last exit code) =" || echo "agente não carregado"
  "$BIN" status
  ;;
logs)
  tail -n 50 -f "$LOG"
  ;;
claude)
  mkdir -p "$(dirname "$MCP_BIN")" "$HOME/.claude/skills"
  (cd "$REPO" && go build -o "$MCP_BIN" ./cmd/omni-mcp)
  claude mcp remove -s user omni >/dev/null 2>&1 || true
  claude mcp add -s user omni -e OMNI_WA_READONLY=on -- "$MCP_BIN" -enable whatsapp
  rm -rf "$HOME/.claude/skills/$SKILL"
  cp -R "$REPO/OS-Optimizer-Playbook/skills/$SKILL" "$HOME/.claude/skills/$SKILL"
  echo "omni-mcp e skill $SKILL instalados — valem nas próximas sessões do Claude Code"
  ;;
*)
  echo "uso: $0 install|uninstall|status|logs|claude" >&2
  exit 2
  ;;
esac
