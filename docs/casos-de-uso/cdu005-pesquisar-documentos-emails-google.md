# CDU005 — Pesquisar Documentos e E-mails no Google Workspace

- **Atores:** Agente de IA / Operador
- **Componentes:** node-conn-google, node-ext-google, node-core-hub

- **Complexidade:** medium
- **Horas Estimadas:** 16
- **Prioridade:** Média
- **Status:** implemented
- **Requisitos:** RF008, RNF002, RNF004

## Descrição

Permite ao agente de IA pesquisar e-mails no Gmail e localizar documentos ou planilhas no Google Drive, extraindo apenas os trechos pertinentes e protegendo a janela de contexto de conteúdos HTML inflados.

## Pré-condições

- Fluxo OAuth2 concluído e arquivo de token armazenado em `~/.config/omni-connectors/google_token.json`.
- Escopos concedidos para leitura/envio de Gmail e leitura de arquivos do Google Drive.

## Pós-condições

- O agente recebe metadados essenciais e texto limpo dos e-mails ou arquivos pesquisados.

## Fluxo Principal

1. O agente invoca `gmail_search_messages` com query no padrão de busca do Gmail (ex: `from:cliente import:true newer_than:7d`).
2. O conector Google executa a busca via `google.golang.org/api/gmail/v1`.
3. O conector recupera as mensagens correspondentes e elimina código HTML, scripts, estilos CSS e headers desnecessários.
4. É retornado um DTO contendo apenas remetente, assunto, data e corpo em texto puro sumarizado.
5. Em consultas ao Google Drive (`drive_search_files`), o conector lista os arquivos com nome, ID, link e data de modificação.

## Fluxos Alternativos

- **Envio de E-mail:** Invocação de `gmail_send_email` monta a mensagem MIME codificada em base64 e despacha via API do Gmail, retornando o ID do e-mail enviado.

## Exceções

- **Token Expirado:** O conector utiliza o `oauth2.TokenSource` para renovar automaticamente o access token; se o refresh token falhar, solicita novo consentimento ao usuário.

## Regras de Negócio

- O corpo de e-mails extensos deve ser truncado em no máximo 3.000 caracteres por mensagem, fornecendo aviso de truncamento se necessário.
- Nenhum token OAuth2 deve ser exposto nos logs ou respostas das ferramentas.

## Critérios de Aceite

- **Given** token de autenticação Google válido **When** invocada `gmail_search_messages` com uma consulta **Then** retorna lista de e-mails em texto puro sem tags HTML.
- **Given** uma requisição de busca no Drive **When** executada com sucesso **Then** retorna IDs e títulos dos arquivos encontrados em formato JSON conciso.
