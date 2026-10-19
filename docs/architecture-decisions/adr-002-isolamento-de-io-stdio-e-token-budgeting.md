# ADR-002 — Isolamento Estrito de I/O em Stdio e Token Budgeting para LLMs

- **Status:** Aceito
- **Data:** 2026-10-18

## Contexto

Ambientes de execução de LLMs operam com restrições severas:
1. **Corrupção de Transporte Stdio:** Em clientes MCP como Claude Code e Antigravity CLI rodando via stdio, o processo filho se comunica com o processo pai exclusivamente via `os.Stdout` usando frames JSON-RPC delimitados. Qualquer linha impressa acidentalmente no `stdout` (por exemplo, `log.Println` ou logs de bibliotecas de terceiros como `whatsmeow` ou drivers HTTP) corrompe o parser do cliente e derruba a sessão.
2. **Degradação de Atenção e Custo de Tokens (Attention Rot):** Retornar payloads brutos de APIs (ex: HTML completo de emails do Gmail, JSON com centenas de campos do Slack ou transcrições inteiras sem formatação) satura a janela de contexto da LLM, aumenta custos e reduz dramaticamente a acurácia das respostas.

## Decisão

1. **Roteamento de Logs Exclusivo para os.Stderr:**
   - O logger padrão em Go (`slog`) é inicializado no `main.go` direcionando exclusivamente para `os.Stderr`:
     `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))`.
   - Todos os conectores e pacotes internos são terminantemente proibidos de usar `fmt.Print*` ou `os.Stdout`.
2. **Política de Token Budgeting e DTOs Enxutos:**
   - Nenhum conector deve expor estruturas brutas de APIs externas.
   - Cada conector deve mapear os resultados para DTOs minimizados contendo apenas campos relevantes para raciocínio da IA (ex: `id`, `sender`, `date`, `snippet`).
   - Textos longos devem ser truncados em limites predefinidos (ex: 2.000 a 4.000 caracteres) com indicador explícito de corte `[... restante truncado ...]`.
3. **Ativação Seletiva de Conectores:**
   - Adição da flag `--enable=mod1,mod2` e da variável de ambiente `ENABLED_CONNECTORS` para inicializar somente os módulos demandados na tarefa.

## Consequências

- **Positivo:** Sessões MCP 100% estáveis sem falhas de parse JSON-RPC no stdio.
- **Positivo:** Preservação máxima da janela de contexto da LLM e velocidade superior de processamento.
- **Negativo:** Exige criação e manutenção de DTOs específicos em cada adapter de conector, em vez de repassar a resposta da API externa diretamente.
