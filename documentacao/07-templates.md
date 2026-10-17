# Modulo: templates

**Diretorio:** `templates/`

## Responsabilidade

Templates HTML renderizados no servidor via `html/template`. Cada pagina e composta pelo layout base (`base.html`) + conteudo especifico.

## Layout base (base.html)

Contem:
- **Sidebar** — navegacao principal com links para todas as telas
- **Header** — titulo da pagina + botoes de acao
- **Tema** — toggle dark/light mode (salvo em localStorage)
- **Ocultar valores** — botao olho que esconde todos os valores monetarios
- **Mobile** (<= 960px) — sidebar em overlay; barra superior fixa para o botao ☰ (o conteudo comeca abaixo dela); tabelas `.table-responsive` viram cartoes (<= 640px) com checkbox em coluna propria e acoes na lateral
- **Menu do usuario** — 🔔 Notificacoes e Sair ao lado do nome
- **Loading** — `<dialog id="page-loading">` aberto com `showModal()` (camada mais alta do navegador, acima do menu e de outros modais) ao navegar ou enviar formularios, apos 150ms; botoes de acoes HTMX ganham `aria-busy` (spinner) e ficam bloqueados. Links de download usam `data-no-loading`. API em JS: `FinLoading.mostrar()` / `FinLoading.esconder()`
- **Icones compartilhados** — `{{template "ico-telegram"}}` e `{{template "ico-email"}}`
- **JavaScript utilitario:**
  - `initTableSort(tableId)` — ordena tabela ao clicar no cabecalho
  - `initSelecao(tableId)` — checkboxes com barra flutuante de soma
  - `initTableFilter({...})` — filtros por busca e selects
  - `populateSelectFromCol(selectId, tableId, col)` — popula select a partir de valores da tabela
  - Dialog helpers (`data-open-dialog`, `data-close-dialog`)

## Templates de pagina

| Template | Rota | Descricao |
|----------|------|-----------|
| `login.html` | /login | Tela de login |
| `dashboard.html` | / | Dashboard com cards, graficos e tabelas. Cada widget e um bloco `{{define "dw-<chave>"}}`, exibido na ordem/coluna do usuario; o dialogo ⚙️ Widgets tem ↑ ↓ e escolha de coluna (o Resumo fica fixo no topo) |
| `despesas.html` | /despesas | Despesas fixas + parcelamentos (tabs) |
| `pagamentos.html` | /pagamentos | Pagamentos do mes + janela "📱 WhatsApp" (previa em balao, copiar, enviar no WhatsApp, enviar por Telegram ou e-mail) |
| `notificacoes.html` | /notificacoes | Envio mensal das contas: e-mail/Telegram, dia, mes atual/seguinte, Chat ID, ajuda (!) do Telegram e "Enviar agora" |
| `receitas.html` | /receitas | Lista de receitas com edicao inline |
| `investimentos.html` | /investimentos | Investimentos + reserva de emergencia |
| `emprestimos.html` | /emprestimos | Emprestimos com pagamentos parciais |
| `planejamento.html` | /planejamento | Planejamento FIRE |
| `cadastros.html` | /cadastros | Tabs: categorias, cartoes, integracao bancaria |
| `transacoes.html` | /transacoes | Transacoes bancarias com filtros e acoes |
| `minha-familia.html` | /minha-familia | Gerenciamento de membros da familia |
| `usuarios.html` | /usuarios | Admin: gerenciar usuarios |
| `familias.html` | /familias | Admin: gerenciar familias |
| `integracoes.html` | /integracoes | Configuracao de integracoes (BCB, cotacoes, Telegram, Feriados, Google Sheets, Pluggy); botao (!) com "para que serve" e "como configurar" em cada uma; segredos nunca sao exibidos |
| `forgot-password.html` | /forgot-password | Recuperacao de senha (standalone, sem sidebar) |
| `reset-password.html` | /reset-senha | Redefinir senha via token (standalone) |
| `importexport.html` | /dados | Import/export XLSX |

## Templates de relatorio

Layout separado (`relatorio-base.html`) otimizado para impressao:
- `relatorio-dashboard.html`
- `relatorio-despesas.html`
- `relatorio-receitas.html`
- `relatorio-investimentos.html`
- `relatorio-emprestimos.html`

## Padrao de template

Cada template define um bloco `content`:

```html
{{define "content"}}
<div class="page-header">
  <h2>Titulo da Pagina</h2>
  <button data-open-dialog="dialog-novo">+ Novo</button>
</div>

<!-- conteudo da pagina -->

<dialog id="dialog-novo">
  <!-- formulario -->
</dialog>
{{end}}
```

## Tabs (cadastros.html)

O template de cadastros usa tabs via query string:
- `?tab=categorias` — Categorias (default)
- `?tab=cartoes` — Cartoes
- `?tab=integracao` — Integracao Bancaria (admin only)

A tab de integracao inclui:
1. Form de credenciais Pluggy com badge de status
2. Botao "Conectar Novo Banco" (Pluggy Connect Widget via CDN)
3. Tabela de contas conectadas com acoes (sync/desconectar)

## CSS

O arquivo `static/app.css` contem o design system completo:
- Variaveis CSS em `:root` (cores, espacamentos, bordas)
- Dark mode com glassmorphism
- `.page-actions` para botoes no header
- Tabelas responsivas com scroll horizontal
- Componentes: cards, badges, dialogs, tabs, filtros

## Tipografia e componentes (app.css)

- **Escala unica de fonte**: `--fs-2xs` (.72rem) · `--fs-xs` · `--fs-sm` · `--fs-md` · `--fs-base` · `--fs-lg` · `--fs-xl` · `--fs-2xl` (1.8rem). Use as variaveis em vez de tamanhos soltos.
- **Base de 15px** em todas as larguras (`--pico-font-size: 93.75%`).
- **Modais**: botao ✕ discreto no cabecalho; botoes do rodape com a mesma altura/peso; no celular ocupam a largura toda.
- **Checkbox/radio/switch marcados** em verde (o fundo geral dos inputs apagava o estado do Pico).
- **Botao de ajuda** `.ajuda-btn` ("!") que abre um `<dialog>` com instrucoes.
