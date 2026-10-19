# CDU006 — Inicializar Hub com Ativação Seletiva

- **Atores:** Operador / Sistema CLI
- **Componentes:** node-core-hub, node-omni-mcp, node-omni-http
- **Complexidade:** low
- **Horas Estimadas:** 8
- **Prioridade:** Alta
- **Status:** implemented
- **Requisitos:** RF009, RNF003

## Descrição

Permite iniciar o servidor MCP ou o gateway HTTP habilitando exclusivamente os conectores desejados para um contexto específico, evitando poluição de contexto e reduzindo o consumo inicial de tokens da LLM.

## Pré-condições

- Parâmetros passados via flag CLI (ex: `--enable=fathom,slack`) ou variável de ambiente (ex: `ENABLED_CONNECTORS=fathom,google`).

## Pós-condições

- O hub inicializa apenas as instâncias de conectores solicitadas.
- O catálogo exposto ao cliente contém estritamente as ferramentas dos módulos habilitados.

## Fluxo Principal

1. O operador ou cliente executa `omni-mcp --enable=fathom,slack`.
2. O sistema lê as flags e variáveis de ambiente.
3. O registro central filtra a lista de conectores disponíveis, instanciando e inicializando apenas os selecionados.
4. O servidor MCP ou HTTP publica unicamente as ferramentas desses módulos.
5. Em consultas de saúde (`GET /v1/health`), a lista de conectores ativos reflete exatamente os módulos em execução.

## Fluxos Alternativos

- **Habilitar Todos os Conectores:** Se a flag `--enable=all` ou nenhuma flag for informada e houver configuração para todos, todos os conectores são inicializados.

## Exceções

- **Conector Inválido Informado:** Se o operador solicitar um módulo desconhecido (ex: `--enable=invalido`), o sistema emite mensagem de aviso em stderr e aborta a inicialização com código de erro.

## Regras de Negócio

- O tempo de inicialização do processo deve ser inferior a 1 segundo para não atrasar a inicialização do cliente MCP via stdio.

## Critérios de Aceite

- **Given** o comando `omni-mcp --enable=fathom` **When** o cliente executa `tools/list` **Then** apenas ferramentas com prefixo `fathom_` são listadas.
- **Given** uma chamada a `GET /v1/health` **When** o gateway foi iniciado com `--enable=slack,mattermost` **Then** o JSON de resposta exibe `enabled_connectors: ["slack", "mattermost"]`.
