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
- **Mobile** — sidebar responsiva com overlay em telas <= 768px
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
| `dashboard.html` | / | Dashboard com cards, graficos e tabelas |
| `despesas.html` | /despesas | Despesas fixas + parcelamentos (tabs) |
| `pagamentos.html` | /pagamentos | Pagamentos do mes |
| `receitas.html` | /receitas | Lista de receitas com edicao inline |
| `investimentos.html` | /investimentos | Investimentos + reserva de emergencia |
| `emprestimos.html` | /emprestimos | Emprestimos com pagamentos parciais |
| `planejamento.html` | /planejamento | Planejamento FIRE |
| `cadastros.html` | /cadastros | Tabs: categorias, cartoes, integracao bancaria |
| `transacoes.html` | /transacoes | Transacoes bancarias com filtros e acoes |
| `minha-familia.html` | /minha-familia | Gerenciamento de membros da familia |
| `usuarios.html` | /usuarios | Admin: gerenciar usuarios |
| `familias.html` | /familias | Admin: gerenciar familias |
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
